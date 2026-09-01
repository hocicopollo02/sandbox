package sandbox

import "github.com/hocicopollo02/sandbox/internal/model"

type Persistence = model.Persistence
type HomeMode = model.HomeMode
type Status = model.Status
type Distribution = model.Distribution
type Record = model.Record
type ListEntry = model.ListEntry
type Info = model.Info
type CreateOptions = model.CreateOptions
type DeleteOptions = model.DeleteOptions

type StopResult string

type CreateResult string

const (
	CreateResultCreated   CreateResult = "created"
	CreateResultUnchanged CreateResult = "unchanged"
	CreateResultRemoved   CreateResult = "removed"

	StopResultStopped   StopResult = "stopped"
	StopResultUnchanged StopResult = "unchanged"
)

const (
	Disposable = model.Disposable
	Persistent = model.Persistent

	IsolatedHome = model.IsolatedHome
	SharedHome   = model.SharedHome

	Running = model.Running
	Stopped = model.Stopped
	Missing = model.Missing
	Unknown = model.Unknown
)

var Distributions = model.Distributions
var FindDistribution = model.FindDistribution
var ValidateName = model.ValidateName
