package graph

import (
	"context"
	"fmt"
	"strconv"

	"github.com/graph-gophers/dataloader"
	"github.com/neilZon/workout-logger-api/database"
	"github.com/neilZon/workout-logger-api/graph/model"
	"github.com/neilZon/workout-logger-api/middleware"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"gorm.io/gorm"
)

// AddExercise is the resolver for the addExercise field.
func (r *mutationResolver) AddExercise(ctx context.Context, workoutSessionID string, exercise model.ExerciseInput) (string, error) {
	u, err := middleware.GetUser(ctx)
	if err != nil {
		return "", err
	}

	userId := fmt.Sprintf("%d", u.ID)
	err = r.ACS.CanAccessWorkoutSession(userId, workoutSessionID)
	if err != nil {
		return "", gqlerror.Errorf("Error Adding Exercise: %s", err.Error())
	}

	// todo: check can access exercise routines that are being added

	var setEntries []database.SetEntry
	for _, s := range exercise.SetEntries {
		setEntries = append(setEntries, database.SetEntry{
			Reps:   uint(s.Reps),
			Weight: float32(s.Weight),
		})
	}

	workoutSessionIDUint, err := strconv.ParseUint(workoutSessionID, 10, 32)
	if err != nil {
		return "", gqlerror.Errorf("Error Adding Exercise: %s", err.Error())
	}

	exerciseRoutineID, err := strconv.ParseUint(exercise.ExerciseRoutineID, 10, 32)
	if err != nil {
		return "", gqlerror.Errorf("Error Adding Exercise: %s", err.Error())
	}

	dbExercise := &database.Exercise{
		WorkoutSessionID:  uint(workoutSessionIDUint),
		ExerciseRoutineID: uint(exerciseRoutineID),
		Sets:              setEntries,
		Notes:             exercise.Notes,
	}

	err = database.AddExercise(r.DB, dbExercise)
	if err != nil {
		return "", gqlerror.Errorf("Error Adding Exercise: %s", err.Error())
	}

	return fmt.Sprintf("%d", dbExercise.ID), nil
}

// Exercise is the resolver for the exercise field.
func (r *queryResolver) Exercise(ctx context.Context, exerciseID string) (*model.Exercise, error) {
	u, err := middleware.GetUser(ctx)
	if err != nil {
		return &model.Exercise{}, err
	}

	exerciseIDUint, err := strconv.ParseUint(exerciseID, 10, 64)
	if err != nil {
		return &model.Exercise{}, gqlerror.Errorf("Error Getting Exercise: Invalid Exercise ID")
	}

	exercise := &database.Exercise{
		Model: gorm.Model{
			ID: uint(exerciseIDUint),
		},
	}
	err = database.GetExercise(r.DB, exercise, false)
	if err != nil {
		return &model.Exercise{}, gqlerror.Errorf("Error Getting Exercise: %s", err.Error())
	}

	err = r.ACS.CanAccessWorkoutSession(fmt.Sprintf("%d", u.ID), fmt.Sprintf("%d", exercise.WorkoutSessionID))
	if err != nil {
		return &model.Exercise{}, gqlerror.Errorf("Error Getting Exercise: %s", err.Error())
	}

	return &model.Exercise{
		ID:    exerciseID,
		Notes: exercise.Notes,
	}, nil
}

// UpdateExercise is the resolver for the updateExercise field.
func (r *mutationResolver) UpdateExercise(ctx context.Context, exerciseID string, exercise model.UpdateExerciseInput) (*model.UpdatedExercise, error) {
	u, err := middleware.GetUser(ctx)
	if err != nil {
		return &model.UpdatedExercise{}, err
	}

	exerciseIDUint, err := strconv.ParseUint(exerciseID, 10, strconv.IntSize)
	dbExercise := database.Exercise{
		Model: gorm.Model{
			ID: uint(exerciseIDUint),
		},
	}
	err = database.GetExercise(r.DB, &dbExercise, false)
	if err != nil {
		return &model.UpdatedExercise{}, gqlerror.Errorf("Error Updating Exercise")
	}

	err = r.ACS.CanAccessWorkoutSession(fmt.Sprintf("%d", u.ID), fmt.Sprintf("%d", dbExercise.WorkoutSessionID))
	if err != nil {
		return &model.UpdatedExercise{}, gqlerror.Errorf("Error Updating Exercise: Access Denied")
	}

	updatedExercise := database.Exercise{
		Notes: exercise.Notes,
	}
	err = database.UpdateExercise(r.DB, exerciseID, &updatedExercise)
	if err != nil {
		return &model.UpdatedExercise{}, gqlerror.Errorf("Error Updating Exercise")
	}

	return &model.UpdatedExercise{
		ID:    exerciseID,
		Notes: updatedExercise.Notes,
	}, nil
}

// DeleteExercise is the resolver for the deleteExercise field.
func (r *mutationResolver) DeleteExercise(ctx context.Context, exerciseID string) (int, error) {
	u, err := middleware.GetUser(ctx)
	if err != nil {
		return 0, err
	}

	exerciseIDUint, err := strconv.ParseUint(exerciseID, 10, strconv.IntSize)
	dbExercise := database.Exercise{
		Model: gorm.Model{
			ID: uint(exerciseIDUint),
		},
	}
	err = database.GetExercise(r.DB, &dbExercise, false)
	if err != nil {
		return 0, gqlerror.Errorf("Error Deleting Exercise")
	}

	err = r.ACS.CanAccessWorkoutSession(fmt.Sprintf("%d", u.ID), fmt.Sprintf("%d", dbExercise.WorkoutSessionID))
	if err != nil {
		return 0, gqlerror.Errorf("Error Deleting Exercise: Access Denied")
	}

	err = database.DeleteExercise(r.DB, exerciseID)
	if err != nil {
		return 0, gqlerror.Errorf("Error Deleting Exercise")
	}

	return 1, nil
}

// Exercises is the resolver for the exercises field.
func (r *workoutSessionResolver) Exercises(ctx context.Context, obj *model.WorkoutSession) ([]*model.Exercise, error) {
	loaders := middleware.GetLoaders(ctx)
	thunk := loaders.ExerciseSliceLoader.Load(ctx, dataloader.StringKey(obj.ID))
	result, err := thunk()
	if err != nil {
		return nil, err
	}
	return result.([]*model.Exercise), nil
}

// PrevExercises is the resolver for the prevExercises field.
func (r *workoutSessionResolver) PrevExercises(ctx context.Context, obj *model.WorkoutSession) ([]*model.Exercise, error) {
	if obj.End == nil {
		return []*model.Exercise{}, nil
	}

	// TODO: 1 + (n+m) issue if we query for all workoutSessions, might need to fix
	dbExercises, err := database.GetPrevExercisesByWorkoutRoutineId(r.DB, obj.WorkoutRoutine.ID, obj.Start)
	if err != nil {
		return []*model.Exercise{}, gqlerror.Errorf("Error Getting Exercises")
	}

	var exercises []*model.Exercise
	for _, e := range dbExercises {
		exercises = append(exercises, &model.Exercise{
			ID:    fmt.Sprintf("%d", e.ID),
			Notes: e.Notes,
		})
	}

	return exercises, nil
}
