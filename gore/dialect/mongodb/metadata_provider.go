package mongodb

import (
	"context"
	"fmt"

	"gore/internal/metadata"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MetadataProvider loads MongoDB index metadata from listIndexes.
type MetadataProvider struct {
	client   *mongo.Client
	database string
}

// NewMetadataProvider creates a MongoDB metadata provider.
func NewMetadataProvider(client *mongo.Client, database string) *MetadataProvider {
	return &MetadataProvider{
		client:   client,
		database: database,
	}
}

// Indexes returns index metadata for a given collection.
// Queries the listIndexes command for MongoDB.
func (p *MetadataProvider) Indexes(ctx context.Context, collection string) ([]metadata.IndexInfo, error) {
	if p.client == nil {
		return nil, fmt.Errorf("mongo client is nil")
	}

	db := p.client.Database(p.database)
	col := db.Collection(collection)

	// Use List to get raw bson documents
	cursor, err := col.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var out []metadata.IndexInfo
	for cursor.Next(ctx) {
		var doc bson.D
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}

		info := metadata.IndexInfo{
			Table: collection,
		}

		// Extract fields from the document
		for _, elem := range doc {
			switch elem.Key {
			case "name":
				if name, ok := elem.Value.(string); ok {
					info.Name = name
				}
			case "unique":
				if unique, ok := elem.Value.(bool); ok && unique {
					info.Unique = true
				}
			case "key":
				// Extract key document (field names)
				if key, ok := elem.Value.(bson.D); ok {
					for _, keyElem := range key {
						info.Columns = append(info.Columns, keyElem.Key)
					}
					info.Method = inferIndexMethod(key)
				}
			}
		}

		out = append(out, info)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

// inferIndexMethod infers the index type from the key document.
func inferIndexMethod(key bson.D) string {
	for _, elem := range key {
		switch v := elem.Value.(type) {
		case int32:
			if v == 1 || v == -1 {
				return "B-tree"
			}
		case string:
			switch v {
			case "text":
				return "text"
			case "2dsphere":
				return "2dsphere"
			case "2d":
				return "2d"
			case "geoHaystack":
				return "geoHaystack"
			case "hashed":
				return "hashed"
			case "wildcard":
				return "wildcard"
			}
		}
	}
	return "B-tree"
}

// Columns returns column metadata for a given collection.
// MongoDB doesn't have a traditional schema, so we derive type information
// from a sample document or use $schema aggregation.
func (p *MetadataProvider) Columns(ctx context.Context, collection string) ([]metadata.ColumnInfo, error) {
	if p.client == nil {
		return nil, fmt.Errorf("mongo client is nil")
	}

	// MongoDB is schemaless - columns would need to be inferred from documents
	// For now, return nil to indicate no schema information is available
	return nil, nil
}

// Tables returns all collections in the current database.
func (p *MetadataProvider) Tables(ctx context.Context) ([]string, error) {
	if p.client == nil {
		return nil, fmt.Errorf("mongo client is nil")
	}

	db := p.client.Database(p.database)

	// List collections, filtering out system collections
	opts := options.ListCollections().SetNameOnly(true)
	cursor, err := db.ListCollections(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tables []string
	for cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		if name, ok := result["name"].(string); ok {
			// Skip system collections
			if len(name) > 0 && name[0] != '_' {
				tables = append(tables, name)
			}
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}

// Find finds documents matching the filter.
func (p *MetadataProvider) Find(ctx context.Context, collection string, filter bson.D, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	if p.client == nil {
		return nil, fmt.Errorf("mongo client is nil")
	}

	db := p.client.Database(p.database)
	col := db.Collection(collection)
	return col.Find(ctx, filter, opts...)
}

// InsertOne inserts a single document.
func (p *MetadataProvider) InsertOne(ctx context.Context, collection string, document interface{}) (*mongo.InsertOneResult, error) {
	if p.client == nil {
		return nil, fmt.Errorf("mongo client is nil")
	}

	db := p.client.Database(p.database)
	col := db.Collection(collection)
	return col.InsertOne(ctx, document)
}

// UpdateOne updates a single document.
func (p *MetadataProvider) UpdateOne(ctx context.Context, collection string, filter bson.D, update interface{}) (*mongo.UpdateResult, error) {
	if p.client == nil {
		return nil, fmt.Errorf("mongo client is nil")
	}

	db := p.client.Database(p.database)
	col := db.Collection(collection)
	return col.UpdateOne(ctx, filter, update)
}

// DeleteOne deletes a single document.
func (p *MetadataProvider) DeleteOne(ctx context.Context, collection string, filter bson.D) (*mongo.DeleteResult, error) {
	if p.client == nil {
		return nil, fmt.Errorf("mongo client is nil")
	}

	db := p.client.Database(p.database)
	col := db.Collection(collection)
	return col.DeleteOne(ctx, filter)
}
