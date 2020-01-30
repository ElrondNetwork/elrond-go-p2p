package p2p

import "errors"

// ErrTopicAlreadyExists signals that a topic already exists
var ErrTopicAlreadyExists = errors.New("topic already exists")

// ErrNilValidator signals that a validator hasn't been set for the required topic
var ErrNilValidator = errors.New("no validator has been set for this topic")

// ErrNilTopic signals that a nil topic has been provided
var ErrNilTopic = errors.New("nil topic")

// ErrTopicValidatorOperationNotSupported signals that an unsupported validator operation occurred
var ErrTopicValidatorOperationNotSupported = errors.New("topic validator operation is not supported")

// ErrInvalidValue signals that an invalid value has been provided
var ErrInvalidValue = errors.New("invalid value")

// ErrPeerDiscoveryProcessAlreadyStarted signals that a peer discovery is already turned on
var ErrPeerDiscoveryProcessAlreadyStarted = errors.New("peer discovery is already turned on")
