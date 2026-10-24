package main

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// A single-document RPC stub exercises the real Firestore client, including
// serialization and transaction callback retries. It is not a Firestore emulator.
type firestoreStub struct {
	pb.UnimplementedFirestoreServer
	mu       sync.Mutex
	doc      *pb.Document
	conflict *pb.Document
	commits  int
}

func (s *firestoreStub) BeginTransaction(context.Context, *pb.BeginTransactionRequest) (*pb.BeginTransactionResponse, error) {
	return &pb.BeginTransactionResponse{Transaction: []byte("transaction")}, nil
}

func (s *firestoreStub) BatchGetDocuments(req *pb.BatchGetDocumentsRequest, stream pb.Firestore_BatchGetDocumentsServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	response := &pb.BatchGetDocumentsResponse{ReadTime: timestamppb.Now()}
	if s.doc == nil {
		response.Result = &pb.BatchGetDocumentsResponse_Missing{Missing: req.Documents[0]}
	} else {
		doc := proto.Clone(s.doc).(*pb.Document)
		doc.Name = req.Documents[0]
		response.Result = &pb.BatchGetDocumentsResponse_Found{Found: doc}
	}
	return stream.Send(response)
}

func (s *firestoreStub) Commit(_ context.Context, req *pb.CommitRequest) (*pb.CommitResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commits++
	if s.commits == 1 && s.conflict != nil {
		s.doc = proto.Clone(s.conflict).(*pb.Document)
		return nil, status.Error(codes.Aborted, "another worker won the claim")
	}
	now := timestamppb.Now()
	response := &pb.CommitResponse{CommitTime: now}
	for _, write := range req.Writes {
		update := write.GetUpdate()
		if s.doc == nil {
			s.doc = &pb.Document{Name: update.Name, Fields: make(map[string]*pb.Value), CreateTime: now}
		}
		for k, v := range update.Fields {
			s.doc.Fields[k] = proto.Clone(v).(*pb.Value)
		}
		for _, transform := range write.UpdateTransforms {
			s.doc.Fields[transform.FieldPath] = &pb.Value{ValueType: &pb.Value_TimestampValue{TimestampValue: now}}
		}
		s.doc.UpdateTime = now
		response.WriteResults = append(response.WriteResults, &pb.WriteResult{UpdateTime: now})
	}
	return response, nil
}

func (s *firestoreStub) Rollback(context.Context, *pb.RollbackRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *firestoreStub) snapshot() *pb.Document {
	s.mu.Lock()
	defer s.mu.Unlock()
	return proto.Clone(s.doc).(*pb.Document)
}

func stubDocument(state string, attempts int64, lease time.Time) *pb.Document {
	return &pb.Document{
		CreateTime: timestamppb.Now(), UpdateTime: timestamppb.Now(),
		Fields: map[string]*pb.Value{
			"status":     {ValueType: &pb.Value_StringValue{StringValue: state}},
			"owner":      {ValueType: &pb.Value_StringValue{StringValue: "other-worker"}},
			"attempts":   {ValueType: &pb.Value_IntegerValue{IntegerValue: attempts}},
			"leaseUntil": {ValueType: &pb.Value_TimestampValue{TimestampValue: timestamppb.New(lease)}},
		},
	}
}

func firestoreTestClient(t *testing.T, stub *firestoreStub) (*firestore.Client, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	pb.RegisterFirestoreServer(server, stub)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	t.Cleanup(func() { _ = listener.Close() })
	conn, err := grpc.NewClient("passthrough:///firestore-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	db, err := firestore.NewClient(ctx, "test-project", option.WithGRPCConn(conn), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, ctx
}

func TestFirestorePersistsClaimsAndOutcomes(t *testing.T) {
	s := &firestoreStub{}
	db, ctx := firestoreTestClient(t, s)
	e := testEntries(1)[0]
	owner, acquired, err := claimArticle(ctx, db, e, newArticles)
	if err != nil || !acquired {
		t.Fatalf("initial claim: acquired=%t err=%v", acquired, err)
	}
	fields := s.snapshot().Fields
	if fields["attempts"].GetIntegerValue() != 1 || fields["owner"].GetStringValue() != owner || fields["status"].GetStringValue() != "processing" {
		t.Fatalf("claim fields: %v", fields)
	}
	before := time.Now()
	if err := finishArticle(ctx, db, e, owner, false); err != nil {
		t.Fatal(err)
	}
	fields = s.snapshot().Fields
	if fields["status"].GetStringValue() != "retry" || fields["owner"].GetStringValue() != "" || fields["leaseUntil"].GetTimestampValue().AsTime().Before(before.Add(retryBaseDelay)) {
		t.Fatalf("retry fields: %v", fields)
	}
	if _, acquired, err := claimArticle(ctx, db, e, retryArticles); err != nil || acquired {
		t.Fatalf("backoff claim: %t, %v", acquired, err)
	}
	s.mu.Lock()
	s.doc.Fields["leaseUntil"] = &pb.Value{ValueType: &pb.Value_TimestampValue{TimestampValue: timestamppb.New(time.Now().Add(-time.Minute))}}
	s.mu.Unlock()
	owner, acquired, err = claimArticle(ctx, db, e, retryArticles)
	if err != nil || !acquired || s.snapshot().Fields["attempts"].GetIntegerValue() != 2 {
		t.Fatalf("retry claim: %t, %v", acquired, err)
	}
	if err := finishArticle(ctx, db, e, "stale-owner", true); err == nil {
		t.Fatal("stale owner was accepted")
	}
	if err := finishArticle(ctx, db, e, owner, true); err != nil {
		t.Fatal(err)
	}
	fields = s.snapshot().Fields
	if fields["status"].GetStringValue() != "published" || fields["attempts"].GetIntegerValue() != 2 || fields["url"].GetStringValue() != e.URL {
		t.Fatalf("published fields: %v", fields)
	}
	if _, acquired, err := claimArticle(ctx, db, e, retryArticles); err != nil || acquired {
		t.Fatalf("published claim: %t, %v", acquired, err)
	}
}

func TestFirestoreTransactionRetryClearsAcquiredFlag(t *testing.T) {
	for _, state := range []string{"published", "processing"} {
		t.Run(state, func(t *testing.T) {
			s := &firestoreStub{conflict: stubDocument(state, 1, time.Now().Add(time.Minute))}
			db, ctx := firestoreTestClient(t, s)
			_, acquired, err := claimArticle(ctx, db, testEntries(1)[0], newArticles)
			if err != nil || acquired {
				t.Fatalf("claim from aborted transaction leaked: acquired=%t err=%v", acquired, err)
			}
			if s.snapshot().Fields["owner"].GetStringValue() != "other-worker" {
				t.Fatal("overwrote concurrent owner's record")
			}
			s.mu.Lock()
			commits := s.commits
			s.mu.Unlock()
			if commits != 2 {
				t.Fatalf("commits=%d, want 2 to exercise callback retry", commits)
			}
		})
	}
}

func TestFirestoreRetiresExpiredFinalAttempt(t *testing.T) {
	s := &firestoreStub{doc: stubDocument("processing", maxArticleAttempts, time.Now().Add(-time.Minute))}
	db, ctx := firestoreTestClient(t, s)
	_, acquired, err := claimArticle(ctx, db, testEntries(1)[0], retryArticles)
	if err != nil || acquired {
		t.Fatalf("exhausted claim: acquired=%t err=%v", acquired, err)
	}
	fields := s.snapshot().Fields
	if fields["status"].GetStringValue() != "failed" || fields["owner"].GetStringValue() != "" || fields["attempts"].GetIntegerValue() != maxArticleAttempts {
		t.Fatalf("exhausted fields: %v", fields)
	}
}

func TestFirestoreLegacyRetryWithoutCounter(t *testing.T) {
	doc := stubDocument("retry", 0, time.Now().Add(-time.Minute))
	delete(doc.Fields, "attempts")
	s := &firestoreStub{doc: doc}
	db, ctx := firestoreTestClient(t, s)
	e := testEntries(1)[0]
	if _, acquired, err := claimArticle(ctx, db, e, newArticles); err != nil || acquired {
		t.Fatalf("legacy record did not yield: %t, %v", acquired, err)
	}
	if _, acquired, err := claimArticle(ctx, db, e, retryArticles); err != nil || !acquired {
		t.Fatalf("legacy retry failed: %t, %v", acquired, err)
	}
	if s.snapshot().Fields["attempts"].GetIntegerValue() != 1 {
		t.Fatal("legacy attempt counter was not persisted")
	}
}
