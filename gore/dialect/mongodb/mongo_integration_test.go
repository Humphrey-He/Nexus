package mongodb_test

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"gore/dialect"
	"gore/dialect/mongodb"
)

const (
	testMongoURI     = "mongodb://localhost:27017"
	testMongoDB     = "gore_test"
	testMongoDBURI  = "mongodb://localhost:27017/gore_test"
)

func newTestMongoClient(t *testing.T) *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(testMongoURI))
	if err != nil {
		t.Fatalf("failed to connect to MongoDB: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("failed to ping MongoDB: %v", err)
	}

	return client
}

func newTestMongoDB(t *testing.T) *mongo.Database {
	client := newTestMongoClient(t)
	return client.Database(testMongoDB)
}

func cleanupCollection(db *mongo.Database, collection string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db.Collection(collection).Drop(ctx)
}

// =============================================================================
// Connection Tests
// =============================================================================

func TestMongoDBConnection(t *testing.T) {
	client := newTestMongoClient(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}()

	var result bson.M
	err := client.Database("admin").RunCommand(context.Background(), bson.D{{Key: "ping", Value: 1}}).Decode(&result)
	if err != nil {
		t.Fatalf("failed to ping: %v", err)
	}
	t.Logf("MongoDB connection successful: %v", result)
}

func TestMongoDBConnectionTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(testMongoURI))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}()

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("failed to ping with timeout: %v", err)
	}
}

// =============================================================================
// MetadataProvider.Indexes Tests
// =============================================================================

func TestMongoDBMetadataProviderIndexes(t *testing.T) {
	client := newTestMongoClient(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}()

	db := client.Database(testMongoDB)
	cleanupCollection(db, "users")

	// Create indexes
	_, err := db.Collection("users").Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetName("idx_name"),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_email"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
	})
	if err != nil {
		t.Fatalf("failed to create indexes: %v", err)
	}

	provider := mongodb.NewMetadataProvider(client, testMongoDB)
	ctx := context.Background()

	indexes, err := provider.Indexes(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get indexes: %v", err)
	}

	if len(indexes) == 0 {
		t.Fatal("expected indexes, got none")
	}

	// Build index map for verification
	indexMap := make(map[string]bool)
	for _, idx := range indexes {
		indexMap[idx.Name] = true
		t.Logf("Index: name=%s, columns=%v, method=%s, unique=%v",
			idx.Name, idx.Columns, idx.Method, idx.Unique)
	}

	// Verify expected indexes exist
	expectedIndexes := []string{"idx_name", "idx_email", "idx_status"}
	for _, name := range expectedIndexes {
		if !indexMap[name] {
			t.Errorf("expected index %s not found", name)
		}
	}
}

func TestMongoDBMetadataProviderIndexesUnique(t *testing.T) {
	client := newTestMongoClient(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}()

	db := client.Database(testMongoDB)
	cleanupCollection(db, "users")

	// Create unique index
	_, err := db.Collection("users").Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("idx_unique_email"),
	})
	if err != nil {
		t.Fatalf("failed to create unique index: %v", err)
	}

	provider := mongodb.NewMetadataProvider(client, testMongoDB)
	ctx := context.Background()

	indexes, err := provider.Indexes(ctx, "users")
	if err != nil {
		t.Fatalf("failed to get indexes: %v", err)
	}

	for _, idx := range indexes {
		if idx.Name == "idx_unique_email" {
			if !idx.Unique {
				t.Error("expected idx_unique_email to be unique")
			}
			t.Logf("Unique index verified: name=%s, unique=%v", idx.Name, idx.Unique)
		}
	}
}

func TestMongoDBMetadataProviderIndexesNonExistentCollection(t *testing.T) {
	client := newTestMongoClient(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}()

	provider := mongodb.NewMetadataProvider(client, testMongoDB)
	ctx := context.Background()

	indexes, err := provider.Indexes(ctx, "non_existent_collection_xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(indexes) != 0 {
		t.Errorf("expected 0 indexes for non-existent collection, got %d", len(indexes))
	}
}

// =============================================================================
// MetadataProvider.Tables (Collections) Tests
// =============================================================================

func TestMongoDBMetadataProviderTables(t *testing.T) {
	client := newTestMongoClient(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}()

	db := client.Database(testMongoDB)
	cleanupCollection(db, "users")
	cleanupCollection(db, "orders")

	// Create test collections
	db.Collection("users").InsertOne(context.Background(), bson.M{"name": "test"})
	db.Collection("orders").InsertOne(context.Background(), bson.M{"total": 100})

	provider := mongodb.NewMetadataProvider(client, testMongoDB)
	ctx := context.Background()

	tables, err := provider.Tables(ctx)
	if err != nil {
		t.Fatalf("failed to get tables: %v", err)
	}

	if len(tables) == 0 {
		t.Fatal("expected tables, got none")
	}

	found := false
	for _, table := range tables {
		t.Logf("Collection: %s", table)
		if table == "users" {
			found = true
		}
	}

	if !found {
		t.Error("expected 'users' collection not found")
	}
}

// =============================================================================
// Dialector BuildSelect Tests
// =============================================================================

func TestMongoDBDialectorBuildSelect(t *testing.T) {
	d := &mongodb.Dialector{}

	tests := []struct {
		name     string
		ast      *dialect.QueryAST
		expected string
	}{
		{
			name:     "basic select",
			ast:      &dialect.QueryAST{Table: "users"},
			expected: "db.users.find({})",
		},
		{
			name:     "select with columns",
			ast:      &dialect.QueryAST{Table: "users", Columns: []string{"id", "name"}},
			expected: "db.users.find({}, { id: 1, name: 1 })",
		},
		{
			name:     "select with where",
			ast:      &dialect.QueryAST{Table: "users", Where: []string{"age: 25"}},
			expected: "db.users.find({ age: 25 })",
		},
		{
			name:     "select with limit",
			ast:      &dialect.QueryAST{Table: "users", Limit: 10},
			expected: "db.users.find({}).limit(10)",
		},
		{
			name:     "select with offset",
			ast:      &dialect.QueryAST{Table: "users", Offset: 20},
			expected: "db.users.find({}).skip(20)",
		},
		{
			name:     "select with order by asc",
			ast:      &dialect.QueryAST{Table: "users", OrderBy: []string{"name"}},
			expected: "db.users.find({}).sort({ name: 1 })",
		},
		{
			name:     "select with order by desc",
			ast:      &dialect.QueryAST{Table: "users", OrderBy: []string{"created_at DESC"}},
			expected: "db.users.find({}).sort({ created_at: -1 })",
		},
		{
			name: "full query",
			ast: &dialect.QueryAST{
				Table:   "users",
				Columns: []string{"id", "name"},
				Where:   []string{"status: 'active'"},
				OrderBy: []string{"created_at DESC"},
				Limit:   10,
				Offset:  5,
			},
			expected: "db.users.find({ status: 'active' }, { id: 1, name: 1 }).sort({ created_at: -1 }).limit(10).skip(5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, _, err := d.BuildSelect(tt.ast)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sqlStr != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, sqlStr)
			}
		})
	}
}

func TestMongoDBDialectorBuildSelectNilAST(t *testing.T) {
	d := &mongodb.Dialector{}
	_, _, err := d.BuildSelect(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestMongoDBDialectorBuildSelectEmptyTable(t *testing.T) {
	d := &mongodb.Dialector{}
	_, _, err := d.BuildSelect(&dialect.QueryAST{Table: ""})
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

// =============================================================================
// Dialector BuildInsert Tests
// =============================================================================

func TestMongoDBDialectorBuildInsert(t *testing.T) {
	d := &mongodb.Dialector{}

	tests := []struct {
		name     string
		ast      *dialect.InsertAST
		expected string
	}{
		{
			name:     "insert single document",
			ast:      &dialect.InsertAST{Table: "users", Columns: []string{"name", "email"}, Values: [][]any{{"John", "john@example.com"}}},
			expected: `db.users.insertOne({ name: John, email: john@example.com })`,
		},
		{
			name:     "insert multiple documents",
			ast:      &dialect.InsertAST{Table: "users", Columns: []string{"name", "email"}, Values: [][]any{{"John", "john@example.com"}, {"Jane", "jane@example.com"}}},
			expected: `db.users.insertMany([{ name: John, email: john@example.com }, { name: Jane, email: jane@example.com }])`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, _, err := d.BuildInsert(tt.ast)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sqlStr != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, sqlStr)
			}
		})
	}
}

func TestMongoDBDialectorBuildInsertNilAST(t *testing.T) {
	d := &mongodb.Dialector{}
	_, _, err := d.BuildInsert(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestMongoDBDialectorBuildInsertEmptyTable(t *testing.T) {
	d := &mongodb.Dialector{}
	_, _, err := d.BuildInsert(&dialect.InsertAST{Table: "", Columns: []string{"name"}})
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

func TestMongoDBDialectorBuildInsertNoColumns(t *testing.T) {
	d := &mongodb.Dialector{}
	_, _, err := d.BuildInsert(&dialect.InsertAST{Table: "users"})
	if err == nil {
		t.Fatal("expected error for no columns")
	}
}

// =============================================================================
// Dialector BuildUpdate Tests
// =============================================================================

func TestMongoDBDialectorBuildUpdate(t *testing.T) {
	d := &mongodb.Dialector{}

	tests := []struct {
		name     string
		ast      *dialect.UpdateAST
		expected string
	}{
		{
			name:     "update single column",
			ast:      &dialect.UpdateAST{Table: "users", Columns: []string{"name"}, Where: []string{"_id: '123'"}},
			expected: "db.users.updateOne({ _id: '123' }, { $set: { name: ? } })",
		},
		{
			name:     "update multiple columns",
			ast:      &dialect.UpdateAST{Table: "users", Columns: []string{"name", "email"}, Where: []string{"_id: '123'"}},
			expected: "db.users.updateOne({ _id: '123' }, { $set: { name: ?, email: ? } })",
		},
		{
			name:     "update without where",
			ast:      &dialect.UpdateAST{Table: "users", Columns: []string{"name"}},
			expected: "db.users.updateOne({}, { $set: { name: ? } })",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, _, err := d.BuildUpdate(tt.ast)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sqlStr != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, sqlStr)
			}
		})
	}
}

func TestMongoDBDialectorBuildUpdateNilAST(t *testing.T) {
	d := &mongodb.Dialector{}
	_, _, err := d.BuildUpdate(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestMongoDBDialectorBuildUpdateEmptyTable(t *testing.T) {
	d := &mongodb.Dialector{}
	_, _, err := d.BuildUpdate(&dialect.UpdateAST{Table: "", Columns: []string{"name"}})
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

func TestMongoDBDialectorBuildUpdateNoColumns(t *testing.T) {
	d := &mongodb.Dialector{}
	_, _, err := d.BuildUpdate(&dialect.UpdateAST{Table: "users"})
	if err == nil {
		t.Fatal("expected error for no columns")
	}
}

// =============================================================================
// Dialector BuildDelete Tests
// =============================================================================

func TestMongoDBDialectorBuildDelete(t *testing.T) {
	d := &mongodb.Dialector{}

	tests := []struct {
		name     string
		ast      *dialect.DeleteAST
		expected string
	}{
		{
			name:     "delete with where",
			ast:      &dialect.DeleteAST{Table: "users", Where: []string{"_id: '123'"}},
			expected: "db.users.deleteOne({ _id: '123' })",
		},
		{
			name:     "delete without where",
			ast:      &dialect.DeleteAST{Table: "users"},
			expected: "db.users.deleteOne({})",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, _, err := d.BuildDelete(tt.ast)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sqlStr != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, sqlStr)
			}
		})
	}
}

func TestMongoDBDialectorBuildDeleteNilAST(t *testing.T) {
	d := &mongodb.Dialector{}
	_, _, err := d.BuildDelete(nil)
	if err == nil {
		t.Fatal("expected error for nil ast")
	}
}

func TestMongoDBDialectorBuildDeleteEmptyTable(t *testing.T) {
	d := &mongodb.Dialector{}
	_, _, err := d.BuildDelete(&dialect.DeleteAST{Table: ""})
	if err == nil {
		t.Fatal("expected error for empty table")
	}
}

// =============================================================================
// Real Query Execution Tests
// =============================================================================

func TestMongoDBRealInsertAndFind(t *testing.T) {
	db := newTestMongoDB(t)
	cleanupCollection(db, "users")

	// Insert test data directly
	_, err := db.Collection("users").InsertOne(context.Background(), bson.M{
		"name":   "TestUser",
		"email":  "testuser@test.example.com",
		"status": 1,
	})
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	// Query using gore-generated query syntax
	d := &mongodb.Dialector{}
	ast := &dialect.QueryAST{
		Table:   "users",
		Columns: []string{"name", "email"},
		Where:   []string{`email: "testuser@test.example.com"`},
	}

	queryStr, _, err := d.BuildSelect(ast)
	if err != nil {
		t.Fatalf("failed to build select: %v", err)
	}
	t.Logf("Generated query: %s", queryStr)

	// Note: MongoDB Go driver uses native Go query, not string
	// The gore dialect generates JavaScript-style query strings for reference
	// For actual execution, one would use the MetadataProvider.Find method
	cursor, err := db.Collection("users").Find(context.Background(), bson.D{{Key: "email", Value: "testuser@test.example.com"}})
	if err != nil {
		t.Fatalf("failed to find: %v", err)
	}
	defer cursor.Close(context.Background())

	var results []bson.M
	if err := cursor.All(context.Background(), &results); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 user, got %d", len(results))
	}
}

func TestMongoDBRealUpdateAndFind(t *testing.T) {
	db := newTestMongoDB(t)
	cleanupCollection(db, "users")

	// Setup: insert test user
	db.Collection("users").InsertOne(context.Background(), bson.M{
		"name":   "UpdateTest",
		"email":  "updatetest@test.example.com",
		"status": 1,
	})

	// Update
	filter := bson.D{{Key: "email", Value: "updatetest@test.example.com"}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: "UpdatedName"}, {Key: "status", Value: 2}}}}
	_, err := db.Collection("users").UpdateOne(context.Background(), filter, update)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Verify update
	var result bson.M
	err = db.Collection("users").FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		t.Fatalf("failed to find updated: %v", err)
	}

	if result["name"] != "UpdatedName" {
		t.Errorf("expected name 'UpdatedName', got %v", result["name"])
	}
	if result["status"] != int32(2) {
		t.Errorf("expected status 2, got %v", result["status"])
	}
}

func TestMongoDBRealDeleteAndFind(t *testing.T) {
	db := newTestMongoDB(t)
	cleanupCollection(db, "users")

	// Setup: insert test user
	db.Collection("users").InsertOne(context.Background(), bson.M{
		"name":   "DeleteTest",
		"email":  "deletetest@test.example.com",
		"status": 1,
	})

	// Delete
	filter := bson.D{{Key: "email", Value: "deletetest@test.example.com"}}
	_, err := db.Collection("users").DeleteOne(context.Background(), filter)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify delete
	count, err := db.Collection("users").CountDocuments(context.Background(), filter)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 documents after delete, got %d", count)
	}
}

func TestMongoDBRealBatchInsert(t *testing.T) {
	db := newTestMongoDB(t)
	cleanupCollection(db, "users")

	// Batch insert
	docs := []interface{}{
		bson.M{"name": "User1", "email": "user1@test.example.com", "status": 1},
		bson.M{"name": "User2", "email": "user2@test.example.com", "status": 1},
		bson.M{"name": "User3", "email": "user3@test.example.com", "status": 1},
	}
	_, err := db.Collection("users").InsertMany(context.Background(), docs)
	if err != nil {
		t.Fatalf("failed to batch insert: %v", err)
	}

	// Verify count
	count, err := db.Collection("users").CountDocuments(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 users, got %d", count)
	}
}

// =============================================================================
// MetadataProvider Direct Operations Tests
// =============================================================================

func TestMongoDBMetadataProviderFind(t *testing.T) {
	db := newTestMongoDB(t)
	cleanupCollection(db, "users")

	// Insert test data
	db.Collection("users").InsertOne(context.Background(), bson.M{
		"name":   "FindTest",
		"email":  "findtest@test.example.com",
		"status": 1,
	})

	client := newTestMongoClient(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}()

	provider := mongodb.NewMetadataProvider(client, testMongoDB)
	ctx := context.Background()

	cursor, err := provider.Find(ctx, "users", bson.D{{Key: "email", Value: "findtest@test.example.com"}})
	if err != nil {
		t.Fatalf("failed to find: %v", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestMongoDBMetadataProviderInsertOne(t *testing.T) {
	client := newTestMongoClient(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}()

	db := client.Database(testMongoDB)
	cleanupCollection(db, "users")

	provider := mongodb.NewMetadataProvider(client, testMongoDB)
	ctx := context.Background()

	result, err := provider.InsertOne(ctx, "users", bson.M{
		"name":   "InsertTest",
		"email":  "inserttest@example.com",
		"status": 1,
	})
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	if result.InsertedID == nil {
		t.Error("expected InsertedID to be set")
	} else {
		t.Logf("Inserted document ID: %v", result.InsertedID)
	}
}

func TestMongoDBMetadataProviderUpdateOne(t *testing.T) {
	client := newTestMongoClient(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}()

	db := client.Database(testMongoDB)
	cleanupCollection(db, "users")

	// Insert first
	db.Collection("users").InsertOne(context.Background(), bson.M{
		"name":   "UpdateTest",
		"email":  "updatetest@example.com",
		"status": 1,
	})

	provider := mongodb.NewMetadataProvider(client, testMongoDB)
	ctx := context.Background()

	result, err := provider.UpdateOne(ctx, "users",
		bson.D{{Key: "email", Value: "updatetest@example.com"}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: "UpdatedName"}, {Key: "status", Value: 2}}}})
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	if result.ModifiedCount != 1 {
		t.Errorf("expected 1 modified, got %d", result.ModifiedCount)
	}
}

func TestMongoDBMetadataProviderDeleteOne(t *testing.T) {
	client := newTestMongoClient(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Disconnect(ctx)
	}()

	db := client.Database(testMongoDB)
	cleanupCollection(db, "users")

	// Insert first
	db.Collection("users").InsertOne(context.Background(), bson.M{
		"name":   "DeleteTest",
		"email":  "deletetest@example.com",
		"status": 1,
	})

	provider := mongodb.NewMetadataProvider(client, testMongoDB)
	ctx := context.Background()

	result, err := provider.DeleteOne(ctx, "users", bson.D{{Key: "email", Value: "deletetest@example.com"}})
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	if result.DeletedCount != 1 {
		t.Errorf("expected 1 deleted, got %d", result.DeletedCount)
	}
}
