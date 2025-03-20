/*
Package iterators provides a set of iterators for various data structures.

# Elements of slices as pointers

The main motivation is making it easier to work with slices of large structures, where people often choose an iterator
pattern, which makes a shallow copy of each struct in the slice:

	var myStructs []MyStruct
	for _, myStruct := range myStructs {
		// myStruct is a shallow *COPY* of each element - any changes in the loop have no effect on the slice element.
	}

In order to improve that, and also not have to use the throw-away `_` variable, [SlicePointerValues] may be used to
iterate across pointers to each element in the slice:

	var myStructs []MyStruct
	for myStructPtr := range iterators.SlicePointerValues(myStructs) {
		// myStructPtr is a pointer to each element - any changes in the loop will be reflected in the slice element.
	}

If you still need the index, you can use [SlicePointers]:

	var myStructs []MyStruct
	for i, myStructPtr := range iterators.SlicePointers(myStructs) {
		// myStructPtr is a pointer to each element - any changes in the loop will be reflected in the slice element.
		// i is the index of the element.
	}

# Walking json

The https://github.com/tidwall/gjson library is already included as a dependency. If you need to walk a json document,
the [WalkGjsonLeaves] iterator is available, which will yield the json paths to all leaves in the document along with
the [gjson.Result] for each leaf.
*/
package iterators
