# GX: a Programming Language for Machine Learning

GX is a domain-specialised programming language for machine learning. GX uses the Go syntax with a few extensions and the Go type system. The main extension is that GX includes symbolic axis lengths for arrays.

Some links to learn how to program in GX:
* GX test files: see the .gx files in third_party/gxlang/gx/build/builder/testdata and its subfolders
* Gemma implementations in GX: third_party/gxlang/gemma/gemma/gemma.gx and third_party/gxlang/gemma/gemma/gemma3.gx
* GX standard library: third_party/gxlang/gx/stdlib

GX provides multiple backends:
* Pure go backend: third_party/gxlang/gx/golang/backend/backend.go
* XLA backend: third_party/gxlang/xlapjrt/plugin/plugin.go
* Tensorflow backend: third_party/gxlang/pygx/backends/tensorflow

GX provides bindings for multiple languages:
* Go bindings: third_party/gxlang/gx/golang/binder/gobindings. Example: third_party/gxlang/gemma/gemma.go.
* C++ bindings: third_party/gxlang/gx/golang/binder/ccbindings. Example: gdm/gxlang/examples/helloworld/helloworld_cc_test.cc.
* Python bindings: third_party/gxlang/pygx. Example: third_party/gxlang/pygx/dyngx_test.py
