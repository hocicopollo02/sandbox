package sandbox

import "github.com/pablo/sandbox/internal/model"

type Persistence = model.Persistence
type HomeMode = model.HomeMode
type Status = model.Status
type Distribution = model.Distribution
type Record = model.Record
type ListEntry = model.ListEntry
type Info = model.Info
type CreateOptions = model.CreateOptions
type DeleteOptions = model.DeleteOptions

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
