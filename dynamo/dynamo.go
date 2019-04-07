package dynamo

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dyna "github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/dynamodbiface"
	"github.com/eriktate/docshelf"
	"github.com/eriktate/docshelf/env"
)

const (
	defUserTable  = "ds_user"
	defDocTable   = "ds_doc"
	defTagTable   = "ds_tag"
	defGroupTable = "ds_group"
)

// A Store has methods that know how to interact with docshelf data in Dynamo.
type Store struct {
	client dynamodbiface.DynamoDBAPI
	fs     docshelf.FileStore

	userTable  string
	docTable   string
	tagTable   string
	groupTable string
}

// New creates a new Store struct.
func New(fs docshelf.FileStore) (Store, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return Store{}, err
	}

	cfg.Region = endpoints.UsEast1RegionID
	svc := dynamodb.New(cfg)

	store := Store{
		client:     svc,
		fs:         fs,
		userTable:  env.GetEnvString("DS_DYNAMO_USER_TABLE", defUserTable),
		docTable:   env.GetEnvString("DS_DYNAMO_DOC_TABLE", defDocTable),
		tagTable:   env.GetEnvString("DS_DYNAMO_TAG_TABLE", defTagTable),
		groupTable: env.GetEnvString("DS_DYNAMO_GROUP_TABLE", defGroupTable),
	}

	return store, store.ensureTables()
}

func makeKey(key string, value interface{}) (map[string]dynamodb.AttributeValue, error) {
	v, err := dyna.Marshal(value)
	if err != nil {
		return nil, err
	}

	return map[string]dynamodb.AttributeValue{key: *v}, nil
}

func (s Store) getItem(ctx context.Context, table, keyName, key string, out interface{}) error {
	k, err := makeKey(keyName, key)
	if err != nil {
		return err
	}

	input := dynamodb.GetItemInput{
		TableName: aws.String(table),
		Key:       k,
	}

	res, err := s.client.GetItemRequest(&input).Send()
	if err != nil {
		return err
	}

	if err := dyna.UnmarshalMap(res.Item, out); err != nil {
		return err
	}

	return nil
}

// ensureTables concurrently ensures dynamo tables are created. Doing this in parallel
// significantly reduces the wait time for dynamo to be bootstrapped.
func (s Store) ensureTables() error {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	var ensureErr error
	go func() {
		defer wg.Done()
		if err := ensureTable(s.client, s.userTable, userTableInput(s.userTable)); err != nil {
			ensureErr = err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ensureTable(s.client, s.docTable, docTableInput(s.docTable)); err != nil {
			ensureErr = err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ensureTable(s.client, s.tagTable, tagTableInput(s.tagTable)); err != nil {
			ensureErr = err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ensureTable(s.client, s.groupTable, groupTableInput(s.groupTable)); err != nil {
			ensureErr = err
		}
	}()

	wg.Wait()
	return ensureErr
}

func ensureTable(svc dynamodbiface.DynamoDBAPI, table string, input dynamodb.CreateTableInput) error {
	describe := dynamodb.DescribeTableInput{
		TableName: aws.String(table),
	}

	if _, err := svc.DescribeTableRequest(&describe).Send(); err != nil {
		// TODO (erik): This seems like a really fragile err check. Need to find a better way
		// to do this.
		if strings.Contains(err.Error(), dynamodb.ErrCodeResourceNotFoundException) {
			if _, err := svc.CreateTableRequest(&input).Send(); err != nil {
				return err
			}

			if err := svc.WaitUntilTableExists(&describe); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func makeAttrDef(key string, attrType dynamodb.ScalarAttributeType) dynamodb.AttributeDefinition {
	return dynamodb.AttributeDefinition{
		AttributeType: attrType,
		AttributeName: aws.String(key),
	}
}

func userTableInput(userTable string) dynamodb.CreateTableInput {
	gsiKey := dynamodb.KeySchemaElement{
		AttributeName: aws.String("email"),
		KeyType:       dynamodb.KeyTypeHash,
	}

	gsi := dynamodb.GlobalSecondaryIndex{
		IndexName:  aws.String(fmt.Sprintf("%s_email_idx", userTable)),
		KeySchema:  []dynamodb.KeySchemaElement{gsiKey},
		Projection: &dynamodb.Projection{ProjectionType: dynamodb.ProjectionTypeKeysOnly},
	}

	hashKey := dynamodb.KeySchemaElement{
		AttributeName: aws.String("id"),
		KeyType:       dynamodb.KeyTypeHash,
	}

	attrDef := []dynamodb.AttributeDefinition{
		makeAttrDef("id", dynamodb.ScalarAttributeTypeS),
		makeAttrDef("email", dynamodb.ScalarAttributeTypeS),
	}

	return dynamodb.CreateTableInput{
		TableName:              aws.String(userTable),
		BillingMode:            dynamodb.BillingModePayPerRequest,
		AttributeDefinitions:   attrDef,
		GlobalSecondaryIndexes: []dynamodb.GlobalSecondaryIndex{gsi},
		KeySchema:              []dynamodb.KeySchemaElement{hashKey},
	}
}

func docTableInput(docTable string) dynamodb.CreateTableInput {
	hashKey := dynamodb.KeySchemaElement{
		AttributeName: aws.String("path"),
		KeyType:       dynamodb.KeyTypeHash,
	}

	attrDef := []dynamodb.AttributeDefinition{
		makeAttrDef("path", dynamodb.ScalarAttributeTypeS),
	}

	return dynamodb.CreateTableInput{
		TableName:            aws.String(docTable),
		BillingMode:          dynamodb.BillingModePayPerRequest,
		AttributeDefinitions: attrDef,
		KeySchema:            []dynamodb.KeySchemaElement{hashKey},
	}
}

func tagTableInput(tagTable string) dynamodb.CreateTableInput {
	hashKey := dynamodb.KeySchemaElement{
		AttributeName: aws.String("tag"),
		KeyType:       dynamodb.KeyTypeHash,
	}

	attrDef := []dynamodb.AttributeDefinition{
		makeAttrDef("tag", dynamodb.ScalarAttributeTypeS),
	}

	return dynamodb.CreateTableInput{
		TableName:            aws.String(tagTable),
		BillingMode:          dynamodb.BillingModePayPerRequest,
		AttributeDefinitions: attrDef,
		KeySchema:            []dynamodb.KeySchemaElement{hashKey},
	}
}

func groupTableInput(groupTable string) dynamodb.CreateTableInput {
	hashKey := dynamodb.KeySchemaElement{
		AttributeName: aws.String("id"),
		KeyType:       dynamodb.KeyTypeHash,
	}

	attrDef := []dynamodb.AttributeDefinition{
		makeAttrDef("id", dynamodb.ScalarAttributeTypeS),
	}

	return dynamodb.CreateTableInput{
		TableName:            aws.String(groupTable),
		BillingMode:          dynamodb.BillingModePayPerRequest,
		AttributeDefinitions: attrDef,
		KeySchema:            []dynamodb.KeySchemaElement{hashKey},
	}
}

// TODO (erik): Duplicated code shared with bolt backend. Should probably consolidate.
func intersect(left, right []string) []string {
	intersection := make([]string, 0)
	for _, el := range left {
		if contains(right, el) {
			intersection = append(intersection, el)
		}
	}

	return intersection
}

func contains(slice []string, el string) bool {
	for _, s := range slice {
		if s == el {
			return true
		}
	}

	return false
}