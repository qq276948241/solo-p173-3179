package statemachine

import (
	"project173/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransition_ValidFlows(t *testing.T) {
	t.Run("unpaid -> paid (pay)", func(t *testing.T) {
		next, err := Transition(models.OrderUnpaid, EventPay)
		assert.NoError(t, err)
		assert.Equal(t, models.OrderPaid, next)
	})

	t.Run("unpaid -> canceled (cancel)", func(t *testing.T) {
		next, err := Transition(models.OrderUnpaid, EventCancel)
		assert.NoError(t, err)
		assert.Equal(t, models.OrderCanceled, next)
	})

	t.Run("paid -> checked_in (check_in)", func(t *testing.T) {
		next, err := Transition(models.OrderPaid, EventCheckIn)
		assert.NoError(t, err)
		assert.Equal(t, models.OrderCheckedIn, next)
	})

	t.Run("paid -> canceled (cancel)", func(t *testing.T) {
		next, err := Transition(models.OrderPaid, EventCancel)
		assert.NoError(t, err)
		assert.Equal(t, models.OrderCanceled, next)
	})

	t.Run("checked_in -> checked_out (check_out)", func(t *testing.T) {
		next, err := Transition(models.OrderCheckedIn, EventCheckOut)
		assert.NoError(t, err)
		assert.Equal(t, models.OrderCheckedOut, next)
	})

	t.Run("full happy path", func(t *testing.T) {
		status := models.OrderUnpaid
		status, err := Transition(status, EventPay)
		assert.NoError(t, err)
		assert.Equal(t, models.OrderPaid, status)

		status, err = Transition(status, EventCheckIn)
		assert.NoError(t, err)
		assert.Equal(t, models.OrderCheckedIn, status)

		status, err = Transition(status, EventCheckOut)
		assert.NoError(t, err)
		assert.Equal(t, models.OrderCheckedOut, status)
	})
}

func TestTransition_IllegalTransitions(t *testing.T) {
	tests := []struct {
		name    string
		current models.OrderStatus
		event   OrderEvent
	}{
		{"unpaid -> check_in", models.OrderUnpaid, EventCheckIn},
		{"unpaid -> check_out", models.OrderUnpaid, EventCheckOut},
		{"paid -> check_out", models.OrderPaid, EventCheckOut},
		{"paid -> pay (double pay)", models.OrderPaid, EventPay},
		{"checked_in -> pay", models.OrderCheckedIn, EventPay},
		{"checked_in -> cancel", models.OrderCheckedIn, EventCancel},
		{"checked_out -> any event", models.OrderCheckedOut, EventPay},
		{"checked_out -> cancel", models.OrderCheckedOut, EventCancel},
		{"canceled -> pay", models.OrderCanceled, EventPay},
		{"canceled -> check_in", models.OrderCanceled, EventCheckIn},
		{"canceled -> cancel", models.OrderCanceled, EventCancel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Transition(tt.current, tt.event)
			assert.Error(t, err, "expected error for transition %s --%s--> ", tt.current, tt.event)
			assert.Equal(t, "illegal state transition", err.Error())
		})
	}
}

func TestTransition_InvalidCurrentStatus(t *testing.T) {
	_, err := Transition(models.OrderStatus("bogus"), EventPay)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid current order status")
}

func TestCanTransition(t *testing.T) {
	assert.True(t, CanTransition(models.OrderUnpaid, EventPay))
	assert.True(t, CanTransition(models.OrderUnpaid, EventCancel))
	assert.True(t, CanTransition(models.OrderPaid, EventCheckIn))
	assert.True(t, CanTransition(models.OrderPaid, EventCancel))
	assert.True(t, CanTransition(models.OrderCheckedIn, EventCheckOut))

	assert.False(t, CanTransition(models.OrderUnpaid, EventCheckIn))
	assert.False(t, CanTransition(models.OrderPaid, EventCheckOut))
	assert.False(t, CanTransition(models.OrderCheckedOut, EventPay))
	assert.False(t, CanTransition(models.OrderCanceled, EventPay))
	assert.False(t, CanTransition(models.OrderStatus("unknown"), EventPay))
}

func TestAllowedEvents(t *testing.T) {
	events := AllowedEvents(models.OrderUnpaid)
	assert.ElementsMatch(t, []OrderEvent{EventPay, EventCancel}, events)

	events = AllowedEvents(models.OrderPaid)
	assert.ElementsMatch(t, []OrderEvent{EventCheckIn, EventCancel}, events)

	events = AllowedEvents(models.OrderCheckedIn)
	assert.ElementsMatch(t, []OrderEvent{EventCheckOut}, events)

	events = AllowedEvents(models.OrderCheckedOut)
	assert.Empty(t, events)

	events = AllowedEvents(models.OrderCanceled)
	assert.Empty(t, events)

	events = AllowedEvents(models.OrderStatus("invalid"))
	assert.Nil(t, events)
}
