package axe

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/256dpi/fire/coal"
)

func TestJob(t *testing.T) {
	tester.Clean()

	job, err := Enqueue(tester.Store, nil, "foo", &bson.M{"foo": "bar"}, 0, false)
	assert.NoError(t, err)

	list := *tester.FindAll(&Job{}).(*[]*Job)
	assert.Len(t, list, 1)
	assert.Equal(t, "foo", list[0].Name)
	assert.Equal(t, &bson.M{"foo": "bar"}, decodeRaw(list[0].Data, &bson.M{}))
	assert.Equal(t, StatusEnqueued, list[0].Status)
	assert.NotZero(t, list[0].Created)
	assert.NotZero(t, list[0].Available)
	assert.Zero(t, list[0].Started)
	assert.Zero(t, list[0].Ended)
	assert.Zero(t, list[0].Finished)
	assert.Equal(t, 0, list[0].Attempts)
	assert.Equal(t, bson.M(nil), list[0].Result)
	assert.Equal(t, "", list[0].Reason)

	job, err = Dequeue(tester.Store, job.ID(), time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, "foo", job.Name)
	assert.Equal(t, &bson.M{"foo": "bar"}, decodeRaw(job.Data, &bson.M{}))
	assert.Equal(t, StatusDequeued, job.Status)
	assert.NotZero(t, job.Created)
	assert.NotZero(t, job.Available)
	assert.NotZero(t, job.Started)
	assert.Zero(t, job.Ended)
	assert.Zero(t, job.Finished)
	assert.Equal(t, 1, job.Attempts)
	assert.Equal(t, bson.M(nil), job.Result)
	assert.Equal(t, "", job.Reason)

	err = Complete(tester.Store, job.ID(), bson.M{"bar": "baz"})
	assert.NoError(t, err)

	job = tester.Fetch(&Job{}, job.ID()).(*Job)
	assert.Equal(t, "foo", job.Name)
	assert.Equal(t, &bson.M{"foo": "bar"}, decodeRaw(job.Data, &bson.M{}))
	assert.Equal(t, StatusCompleted, job.Status)
	assert.NotZero(t, job.Created)
	assert.NotZero(t, job.Available)
	assert.NotZero(t, job.Started)
	assert.NotZero(t, job.Ended)
	assert.NotZero(t, job.Finished)
	assert.Equal(t, 1, job.Attempts)
	assert.Equal(t, bson.M{"bar": "baz"}, job.Result)
	assert.Equal(t, "", job.Reason)
}

func TestDelayed(t *testing.T) {
	tester.Clean()

	job, err := Enqueue(tester.Store, nil, "foo", nil, 100*time.Millisecond, false)
	assert.NoError(t, err)

	job2, err := Dequeue(tester.Store, job.ID(), time.Hour)
	assert.NoError(t, err)
	assert.Nil(t, job2)

	time.Sleep(120 * time.Millisecond)

	job2, err = Dequeue(tester.Store, job.ID(), time.Hour)
	assert.NoError(t, err)
	assert.NotNil(t, job2)

	job2, err = Dequeue(tester.Store, job.ID(), time.Hour)
	assert.NoError(t, err)
	assert.Nil(t, job2)
}

func TestTimeout(t *testing.T) {
	tester.Clean()

	job, err := Enqueue(tester.Store, nil, "foo", nil, 0, false)
	assert.NoError(t, err)

	job2, err := Dequeue(tester.Store, job.ID(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, job2)

	job2, err = Dequeue(tester.Store, job.ID(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Nil(t, job2)

	time.Sleep(150 * time.Millisecond)

	job2, err = Dequeue(tester.Store, job.ID(), 100*time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, job2)
}

func TestFailed(t *testing.T) {
	tester.Clean()

	job, err := Enqueue(tester.Store, nil, "foo", nil, 0, false)
	assert.NoError(t, err)

	job, err = Dequeue(tester.Store, job.ID(), time.Hour)
	assert.NoError(t, err)
	assert.NotNil(t, job)

	err = Fail(tester.Store, job.ID(), "some error", 0)
	assert.NoError(t, err)

	job = tester.Fetch(&Job{}, job.ID()).(*Job)
	assert.Equal(t, StatusFailed, job.Status)
	assert.NotZero(t, job.Ended)
	assert.Equal(t, "some error", job.Reason)

	job2, err := Dequeue(tester.Store, job.ID(), time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, job.ID(), job2.ID())
	assert.Equal(t, 2, job2.Attempts)
}

func TestFailedDelayed(t *testing.T) {
	tester.Clean()

	job, err := Enqueue(tester.Store, nil, "foo", nil, 0, false)
	assert.NoError(t, err)

	job, err = Dequeue(tester.Store, job.ID(), time.Hour)
	assert.NoError(t, err)
	assert.NotNil(t, job)

	err = Fail(tester.Store, job.ID(), "some error", 100*time.Millisecond)
	assert.NoError(t, err)

	job = tester.Fetch(&Job{}, job.ID()).(*Job)
	assert.Equal(t, StatusFailed, job.Status)
	assert.NotZero(t, job.Ended)
	assert.Equal(t, "some error", job.Reason)

	job2, err := Dequeue(tester.Store, job.ID(), time.Hour)
	assert.NoError(t, err)
	assert.Nil(t, job2)

	time.Sleep(120 * time.Millisecond)

	job3, err := Dequeue(tester.Store, job.ID(), time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, 2, job3.Attempts)
	assert.Equal(t, "some error", job3.Reason)
}

func TestCancelled(t *testing.T) {
	tester.Clean()

	job, err := Enqueue(tester.Store, nil, "foo", nil, 0, false)
	assert.NoError(t, err)

	job, err = Dequeue(tester.Store, job.ID(), time.Hour)
	assert.NoError(t, err)
	assert.NotNil(t, job)

	err = Cancel(tester.Store, job.ID(), "some reason")
	assert.NoError(t, err)

	job = tester.Fetch(&Job{}, job.ID()).(*Job)
	assert.Equal(t, StatusCancelled, job.Status)
	assert.NotZero(t, job.Ended)
	assert.NotZero(t, job.Finished)
	assert.Equal(t, "some reason", job.Reason)
}

func TestEnqueueExclusive(t *testing.T) {
	tester.Clean()

	job1, err := Enqueue(tester.Store, nil, "foo", bson.M{"foo": "bar"}, 0, true)
	assert.NoError(t, err)
	assert.NotNil(t, job1)

	list := *tester.FindAll(&Job{}).(*[]*Job)
	assert.Len(t, list, 1)
	assert.Equal(t, "foo", list[0].Name)
	assert.Equal(t, &bson.M{"foo": "bar"}, decodeRaw(list[0].Data, &bson.M{}))
	assert.Equal(t, StatusEnqueued, list[0].Status)
	assert.NotZero(t, list[0].Created)
	assert.NotZero(t, list[0].Available)
	assert.Zero(t, list[0].Started)
	assert.Zero(t, list[0].Ended)
	assert.Zero(t, list[0].Finished)
	assert.Equal(t, 0, list[0].Attempts)
	assert.Equal(t, bson.M(nil), list[0].Result)
	assert.Equal(t, "", list[0].Reason)

	job2, err := Enqueue(tester.Store, nil, "foo", bson.M{"foo": "bar"}, 0, true)
	assert.NoError(t, err)
	assert.Nil(t, job2)

	list = *tester.FindAll(&Job{}).(*[]*Job)
	assert.Len(t, list, 1)
	assert.Equal(t, "foo", list[0].Name)
	assert.Equal(t, &bson.M{"foo": "bar"}, decodeRaw(list[0].Data, &bson.M{}))
	assert.Equal(t, StatusEnqueued, list[0].Status)
	assert.NotZero(t, list[0].Created)
	assert.NotZero(t, list[0].Available)
	assert.Zero(t, list[0].Started)
	assert.Zero(t, list[0].Ended)
	assert.Zero(t, list[0].Finished)
	assert.Equal(t, 0, list[0].Attempts)
	assert.Equal(t, bson.M(nil), list[0].Result)
	assert.Equal(t, "", list[0].Reason)

	_, err = Dequeue(tester.Store, job1.ID(), time.Second)
	assert.NoError(t, err)

	err = Complete(tester.Store, job1.ID(), nil)
	assert.NoError(t, err)

	time.Sleep(time.Second)

	job3, err := Enqueue(tester.Store, nil, "foo", bson.M{"foo": "baz"}, 0, true)
	assert.NoError(t, err)
	assert.NotNil(t, job3)

	list = *tester.FindAll(&Job{}).(*[]*Job)
	assert.Len(t, list, 2)
	assert.Equal(t, "foo", list[1].Name)
	assert.Equal(t, &bson.M{"foo": "baz"}, decodeRaw(list[1].Data, &bson.M{}))
	assert.Equal(t, StatusEnqueued, list[1].Status)
	assert.NotZero(t, list[1].Created)
	assert.NotZero(t, list[1].Available)
	assert.Zero(t, list[1].Started)
	assert.Zero(t, list[1].Ended)
	assert.Zero(t, list[1].Finished)
	assert.Equal(t, 0, list[1].Attempts)
	assert.Equal(t, bson.M(nil), list[1].Result)
	assert.Equal(t, "", list[1].Reason)
}

func TestAddJobIndexes(t *testing.T) {
	tester.Clean()

	idx := coal.NewIndexer()
	AddJobIndexes(idx, time.Hour)

	assert.NoError(t, idx.Ensure(tester.Store))
	assert.NoError(t, idx.Ensure(tester.Store))
}
