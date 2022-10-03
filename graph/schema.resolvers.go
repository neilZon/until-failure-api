package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"net/mail"
	"os"

	"github.com/neilZon/workout-logger-api/common/database"
	"github.com/neilZon/workout-logger-api/graph/generated"
	"github.com/neilZon/workout-logger-api/graph/model"
	"github.com/neilZon/workout-logger-api/utils"
	"github.com/neilZon/workout-logger-api/utils/config"
	"github.com/neilZon/workout-logger-api/utils/token"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"golang.org/x/crypto/bcrypt"
)

// Login is the resolver for the login field.
func (r *mutationResolver) Login(ctx context.Context, email *string, password *string) (model.AuthResult, error) {
	if _, err := mail.ParseAddress(*email); err != nil {
		return nil, gqlerror.Errorf("Not a valid email")
	}

	dbUser, err := database.GetUserByEmail(r.DB, *email)
	if err != nil {
		return nil, gqlerror.Errorf(err.Error())
	}
	if dbUser.ID == 0 {
		return nil, gqlerror.Errorf("Email does not exist")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(*password)); err != nil {
		return nil, gqlerror.Errorf("Incorrect Password")
	}
	c := token.Credentials{
		ID:    dbUser.ID,
		Email: dbUser.Email,
		Name:  dbUser.Name,
	}

	refreshToken := token.Sign(c, []byte(os.Getenv("REFRESH_SECRET")), config.REFRESH_TTL)
	accessToken := token.Sign(c, []byte(os.Getenv("ACCESS_SECRET")), config.ACCESS_TTL)

	return model.AuthSuccess{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, nil
}

// Signup is the resolver for the signup field.
func (r *mutationResolver) Signup(ctx context.Context, email *string, name *string, password *string, confirmPassword *string) (model.AuthResult, error) {
	if *password != *confirmPassword {
		return nil, gqlerror.Errorf("Passwords don't match")
	}

	// check strength
	if !utils.IsStrong(*password) {
		return nil, gqlerror.Errorf("Password needs at least 1 number and 8 - 16 characters")
	}

	if _, err := mail.ParseAddress(*email); err != nil {
		return nil, gqlerror.Errorf("Not a valid email")
	}

	dbUser, err := database.GetUserByEmail(r.DB, *email)
	// check if user was found from query
	if dbUser.ID != 0 {
		return nil, gqlerror.Errorf("Email already exists")
	}

	// Hashing the password with the default cost of 10
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	u := database.User{Name: *name, Email: *email, Password: string(hashedPassword)}
	err = r.DB.Create(&u).Error
	if err != nil {
		return nil, gqlerror.Errorf(err.Error())
	}

	c := token.Credentials{
		ID:    u.ID,
		Email: u.Email,
		Name:  u.Name,
	}

	refreshToken := token.Sign(c, []byte(os.Getenv("REFRESH_SECRET")), config.REFRESH_TTL)
	accessToken := token.Sign(c, []byte(os.Getenv("ACCESS_SECRET")), config.ACCESS_TTL)

	return model.AuthSuccess{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}, nil
}

// RefreshAccessToken is the resolver for the refreshAccessToken field.
func (r *mutationResolver) RefreshAccessToken(ctx context.Context, refreshToken *string) (*model.RefreshSuccess, error) {
	// read token from context
	claims, err := token.Decode(*refreshToken, []byte(os.Getenv("REFRESH_SECRET")))
	if err != nil {
		return nil, gqlerror.Errorf("Refresh token invalid")
	}
	credentials := token.ClaimsToStruct(claims)

	accessToken := token.Sign(*credentials, []byte(os.Getenv("ACCESS_SECRET")), config.ACCESS_TTL)

	return &model.RefreshSuccess{
		AccessToken: accessToken,
	}, nil
}

// WorkoutRoutines is the resolver for the workoutRoutines field.
func (r *queryResolver) WorkoutRoutines(ctx context.Context) ([]*model.WorkoutRoutine, error) {
	panic(fmt.Errorf("not implemented: WorkoutRoutines - workoutRoutines"))
}

// ExerciseRoutines is the resolver for the exerciseRoutines field.
func (r *queryResolver) ExerciseRoutines(ctx context.Context) ([]*model.ExerciseRoutine, error) {
	panic(fmt.Errorf("not implemented: ExerciseRoutines - exerciseRoutines"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
