package mapprealloc

// smallSeeds is ranged over in crossfile.go. Declaring it in another file
// pins that the range source resolves across files of the package.
var smallSeeds = []string{"a", "b", "c"}
