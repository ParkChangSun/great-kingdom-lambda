package ddb

import (
	"context"
	"great-kingdom-lambda/lib/vars"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type User struct {
	Id           string
	PasswordHash string `json:"-"`
	RefreshToken string `json:"-"`
	W, L         int
	RecentGames  []RecentGame
}

type RecentGame struct {
	BlueId, OrangeId string
	Result           string
	CreatedDate      string
}

type UserRepository struct {
	client *dynamodb.Client
}

func NewUserRepository() *UserRepository {
	return &UserRepository{client: _client}
}

func UserKey(id string) map[string]types.AttributeValue {
	k, _ := attributevalue.MarshalMap(struct{ Id string }{Id: id})
	return k
}

func (r *UserRepository) Get(ctx context.Context, id string) (User, error) {
	query, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(vars.USER_DYNAMODB),
		Key:       UserKey(id),
	})
	if err != nil {
		return User{}, err
	}

	if query.Item == nil {
		return User{}, &ItemNotFoundError{vars.USER_DYNAMODB, id}
	}

	item := User{}
	err = attributevalue.UnmarshalMap(query.Item, &item)
	return item, err
}

func (r *UserRepository) Put(ctx context.Context, user User) error {
	data, _ := attributevalue.MarshalMap(user)
	_, err := r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(vars.USER_DYNAMODB),
		Item:      data,
	})
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(vars.USER_DYNAMODB),
		Key:       UserKey(id),
	})
	return err
}
