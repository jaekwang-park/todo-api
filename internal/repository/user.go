package repository

import (
	"context"

	"github.com/jaekwang-park/todo-api/internal/model"
)

type UserRepository interface {
	GetOrCreate(ctx context.Context, cognitoSub, email string) (model.User, error)
	GetByCognitoSub(ctx context.Context, cognitoSub string) (model.User, error)
	Update(ctx context.Context, user model.User) (model.User, error)
}
