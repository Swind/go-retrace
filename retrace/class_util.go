package retrace

import "strings"

/*
* Convert an internal class name into an external class name.
* e.g. java/lang/Object -> java.lang.Object
 */
func ExternalClassName(name string) string {
	return strings.ReplaceAll(name, "/", ".")
}
