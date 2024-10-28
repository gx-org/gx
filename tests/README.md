# GX testing

Files with a `.gx` extension are used to test the GX compiler and the XLA
backend.

GX tests implement the following testing procedures. First, all the GX files are
parsed and compiled into the GX intermediate representation (gir). If there are
compiling errors, then errors specified in the comments are removed from the set
of reported errors. If there are remaining errors, then the test has failed and
stops. The remaining errors are reported as test errors. If all errors have been
removed, then the test has succeeded.

If the compiler reported no error then, each exported functions with a `Test`
suffix is interpreted into a XLA compute graph. The compute graph is then
executed. The graph output is converted into an output string. This output
string is compared to the `Want:` section in the comments. The test fails if an
error occurs while building or running the XLA graph or if the string in the
comment does not match the output of the graph.

Note that test functions with expected errors cannot be mixed with functions
expected to be executed with no errors.
