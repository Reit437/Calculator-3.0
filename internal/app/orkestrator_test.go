package orkestrator_test

import (
	"errors"
	"sync"
	"testing"

	ork "github.com/Reit437/Calculator-3.0/internal/app"
	er "github.com/Reit437/Calculator-3.0/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestExpressions_ConcurrentAccess(t *testing.T) {
	initialExpressions := []ork.SubExp{
		{Id: "id1", Status: "solved", Result: "5"},
		{Id: "id2", Status: "not solved", Result: "3 * 4"},
	}

	ork.Id = initialExpressions

	var wg sync.WaitGroup
	wg.Add(2)

	// Concurrent readers
	go func() {
		defer wg.Done()
		result := ork.Expressions()
		assert.Len(t, result, 2)
	}()

	go func() {
		defer wg.Done()
		result := ork.Expressions()
		assert.Equal(t, "id1", result[0].Id)
	}()

	wg.Wait()
}

func TestExpressionByID_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		prep    func()
		id      string
		want    ork.SubExp
		wantErr error
	}{
		{
			name: "empty ID list",
			prep: func() {
				ork.Id = []ork.SubExp{}
				ork.Maxid = 0
			},
			id:      "id1",
			want:    ork.SubExp{},
			wantErr: errors.New(er.ErrNotFound),
		},
		{
			name: "invalid ID format",
			prep: func() {
				ork.Id = []ork.SubExp{{Id: "id1"}}
				ork.Maxid = 1
			},
			id:      "invalid",
			want:    ork.SubExp{},
			wantErr: errors.New(er.ErrNotFound),
		},
		{
			name: "ID out of range",
			prep: func() {
				ork.Id = []ork.SubExp{{Id: "id1"}}
				ork.Maxid = 1
			},
			id:      "id999",
			want:    ork.SubExp{},
			wantErr: errors.New(er.ErrNotFound),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prep()
			got, err := ork.ExpressionByID(tt.id)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTaskf_Concurrent(t *testing.T) {
	ork.Tasks = []ork.Task{
		{Id: "id1"}, {Id: "id2"}, {Id: "id3"},
	}

	var wg sync.WaitGroup
	results := make(chan ork.Task, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- ork.Taskf()
		}()
	}

	wg.Wait()
	close(results)

	var gotTasks []string
	for task := range results {
		gotTasks = append(gotTasks, task.Id)
	}

	assert.ElementsMatch(t, []string{"id1", "id2", "id3"}, gotTasks)
	assert.Empty(t, ork.Tasks)
}
