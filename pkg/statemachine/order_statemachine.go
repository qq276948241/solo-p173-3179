package statemachine

import (
	"errors"
	"fmt"
	"project173/models"
)

type OrderEvent string

const (
	EventPay       OrderEvent = "pay"
	EventCheckIn   OrderEvent = "check_in"
	EventCheckOut  OrderEvent = "check_out"
	EventCancel    OrderEvent = "cancel"
)

var transitions = map[models.OrderStatus]map[OrderEvent]models.OrderStatus{
	models.OrderUnpaid: {
		EventPay:    models.OrderPaid,
		EventCancel: models.OrderCanceled,
	},
	models.OrderPaid: {
		EventCheckIn: models.OrderCheckedIn,
		EventCancel:  models.OrderCanceled,
	},
	models.OrderCheckedIn: {
		EventCheckOut: models.OrderCheckedOut,
	},
	models.OrderCheckedOut: {},
	models.OrderCanceled:   {},
}

func CanTransition(current models.OrderStatus, event OrderEvent) bool {
	events, ok := transitions[current]
	if !ok {
		return false
	}
	_, ok = events[event]
	return ok
}

func Transition(current models.OrderStatus, event OrderEvent) (models.OrderStatus, error) {
	events, ok := transitions[current]
	if !ok {
		return "", fmt.Errorf("invalid current order status: %s", current)
	}
	next, ok := events[event]
	if !ok {
		return "", errors.New("illegal state transition")
	}
	return next, nil
}

func AllowedEvents(current models.OrderStatus) []OrderEvent {
	events, ok := transitions[current]
	if !ok {
		return nil
	}
	result := make([]OrderEvent, 0, len(events))
	for e := range events {
		result = append(result, e)
	}
	return result
}
