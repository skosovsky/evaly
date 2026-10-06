package evaly

const (
	defaultArtifactMaxBytes = 32 << 20
)

const (
	splitMixSecondMultiplier = 0x94d049bb133111eb
	splitMixIncrement        = 0x9e3779b97f4a7c15
	splitMixFirstMultiplier  = 0xbf58476d1ce4e5b9
	splitMixSecondShift      = 27
	splitMixFirstShift       = 30
	splitMixFinalShift       = 31
)

const (
	maxEvidenceDiagnostics = 32
)

const (
	bootstrapLowerTail  = .025
	bootstrapConfidence = .95
	bootstrapUpperTail  = .975
	meanPrecisionBits   = 256
	invalidExitCode     = 3
)

const (
	assessmentWireRevision = 3
)

const (
	experimentWireRevision = 3
)
