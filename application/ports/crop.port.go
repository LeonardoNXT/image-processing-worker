package ports

type Rectangle struct{ X, Y, Width, Height int }
type Reference struct{ Width, Height int }
type Crop func(width, height int, reference Reference) Rectangle
