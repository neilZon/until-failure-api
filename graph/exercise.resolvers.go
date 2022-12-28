package graph

import (
	"context"
	"fmt"
	"strconv"

	"github.com/neilZon/workout-logger-api/database"
	"github.com/neilZon/workout-logger-api/graph/model"
	"github.com/neilZon/workout-logger-api/middleware"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"gorm.io/gorm"
)

// Exercises is the resolver for the exercises field.
func (r *queryResolver) Exercises(ctx context.Context, workoutSessionID string) ([]*model.Exercise, error) {
	exercises, err := r.Resolver.WorkoutSession().Exercises(ctx, &model.WorkoutSession{ID: workoutSessionID})
	if err != nil {
		return []*model.Exercise{}, err
	}
	return exercises, nil
}

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

	err = database.AddExercise(r.DB, dbExercise, workoutSessionID)
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
	err = database.GetExercise(r.DB, exercise)
	if err != nil {
		return &model.Exercise{}, gqlerror.Errorf("Error Getting Exercise: %s", err.Error())
	}

	err = r.ACS.CanAccessWorkoutSession(fmt.Sprintf("%d", u.ID), fmt.Sprintf("%d", exercise.WorkoutSessionID))
	if err != nil {
		return &model.Exercise{}, gqlerror.Errorf("Error Getting Exercise: %s", err.Error())
	}

	var setEntries []*model.SetEntry
	for _, s := range exercise.Sets {
		setEntries = append(setEntries, &model.SetEntry{
			ID:     fmt.Sprintf("%d", s.ID),
			Weight: float64(s.Weight),
			Reps:   int(s.Reps),
		})
	}

	return &model.Exercise{
		ID:                exerciseID,
		Sets:              setEntries,
		Notes:             exercise.Notes,
		ExerciseRoutineID: fmt.Sprintf("%d", exercise.ExerciseRoutineID),
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
	err = database.GetExercise(r.DB, &dbExercise)
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
	err = database.GetExercise(r.DB, &dbExercise)
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
	u, err := middleware.GetUser(ctx)
	if err != nil {
		return []*model.Exercise{}, err
	}

	err = r.ACS.CanAccessWorkoutSession(fmt.Sprintf("%d", u.ID), obj.ID)
	if err != nil {
		return []*model.Exercise{}, gqlerror.Errorf("Error Getting Exercises: Access Denied")
	}

	var dbExercises []database.Exercise
	err = database.GetExercises(r.DB, &dbExercises, obj.ID)
	if err != nil {
		return []*model.Exercise{}, gqlerror.Errorf("Error Getting Exercises")
	}

	var exercises []*model.Exercise
	for _, e := range dbExercises {

		var setEntries []*model.SetEntry
		for _, s := range e.Sets {
			setEntries = append(setEntries, &model.SetEntry{
				ID:     fmt.Sprintf("%d", s.ID),
				Weight: float64(s.Weight),
				Reps:   int(s.Reps),
			})
		}

		exercises = append(exercises, &model.Exercise{
			ID:                fmt.Sprintf("%d", e.ID),
			Sets:              setEntries,
			Notes:             e.Notes,
			ExerciseRoutineID: fmt.Sprintf("%d", e.ExerciseRoutineID),
		})
	}

	return exercises, nil
}
