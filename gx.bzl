# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Blaze extensions for GX."""

load("//third_party/bazel_rules/rules_cc/cc:cc_library.bzl", "cc_library")
load("//tools/build_defs/go:go_binary.bzl", "go_binary")
load("//tools/build_defs/go:go_library.bzl", "go_library")
load("//tools/build_defs/go:go_test.bzl", "go_test")
load("//tools/build_defs/label:def.bzl", "parse_label")

def gx_library(name, srcs, deps = [], tags = [], visibility = None, testonly = False):
    """Build a Gx library.

    Args:
      name: name of the build rule.
      srcs: source files to compile.
      deps: list of GX dependencies.
      tags: forwarded to generated targets
      visibility: standard Blaze attribute, passed through.
      testonly: standard Blaze attribute, passed through.
    """
    go_packager = "//third_party/gxlang/gx/golang/packager"
    go_src = name + "_gx.go"
    gx_files = ""
    for src in srcs:
        gx_files = gx_files + "google3/" + "$(location " + src + "),"
    gx_deps = []
    go_deps = []
    for dep in deps:
        dep_pkg, dep_target = parse_label(dep)

        gx_deps.append("google3/" + dep_pkg + "/" + dep_target + "_gx")

        go_dep_target = dep_target + "_gx"
        go_deps.append("//" + dep_pkg + ":" + go_dep_target)

    # Generate the source code to embed GX source files into a Go library (using embed).
    native.genrule(
        name = name + "_package",
        message = "Packaging " + name,
        visibility = visibility,
        testonly = testonly,
        tags = tags,
        srcs = srcs,
        outs = [go_src],
        tools = [go_packager],
        cmd = (
            "$(location " + go_packager + ")" +
            " --go_package_name=" + name + "_gx" +
            " --gx_import_path=google3/" + native.package_name() + "/" + name +
            " --target_folder=$(@D)" +
            " --target_name=" + go_src +
            " --gx_files=" + gx_files +
            " --gx_deps=" + ",".join(gx_deps)
        ),
    )

    # Generate the Go library that other gx_library can depend on.
    go_library(
        name = name + "_gx",
        srcs = [go_src],
        data = srcs,
        testonly = testonly,
        tags = tags,
        deps = go_deps + [
            "//third_party/golang/errors",
            "//third_party/gxlang/gx/build/builder",
            "//third_party/gxlang/gx/build/importers/embedpkg",
            "//third_party/gxlang/gx/build/importers",
            "//third_party/gxlang/gx/build/ir",
        ],
        embedsrcs = srcs,
        visibility = visibility,
        cgo = 1,
    )

    # Generate the source code for the binary to generate the bindings.
    go_generate_binder = "//gdm/gxlang/binder:generate_binder"
    go_binder_src = name + "_binder_gx.go"
    native.genrule(
        name = name + "_binder_genrule",
        message = "Generating binder for " + name,
        visibility = visibility,
        testonly = testonly,
        tags = tags,
        srcs = [name + "_package"],
        outs = [go_binder_src],
        tools = [go_generate_binder],
        cmd = (
            "$(location " + go_generate_binder + ")" +
            " --target_folder=$(@D)" +
            " --target_name=" + go_binder_src +
            " --go_package_name=" + name + "_gx"
        ),
    )

    # Compile the source into the binding generator binary.
    # This binary can then be called for each language to
    # generate bindings.
    go_binary(
        name = name + "_binder_gx",
        srcs = [go_binder_src],
        data = srcs,
        tags = tags,
        deps = [
            name + "_gx",
            "//base/go:flag",
            "//base/go:google",
            "//base/go:log",
            "//third_party/gxlang/gx/golang/binder",
            "//third_party/gxlang/gx/stdlib",
            "//third_party/gxlang/gx/build/builder",
            "//third_party/gxlang/gx/build/importers/embedpkg",
            "//third_party/gxlang/gx/build/importers",
            "//gdm/gxlang/binder:binder",
        ],
    )

def go_gx_library(name, lib, deps = [], tags = [], visibility = None, testonly = False):
    """Generate the Go bindings for a GX library.

    Args:
      name: name of the build rule.
      lib: GX library to write the bindings for.
      deps: list of go_gx_library dependencies.
      tags: forwarded to generated targets
      visibility: standard Blaze attribute, passed through.
      testonly: standard Blaze attribute, passed through.
    """
    want_suffix = "_go_gx"
    if not name.endswith(want_suffix):
        fail("target %s does not end with %s" % (name, want_suffix))
    target_name = name[:-len(want_suffix)]
    gx_library = target_name + "_gx.go"
    gx_binder = target_name + "_binder_gx"
    go_file = target_name + "_go_gx/" + name + ".go"
    native.genrule(
        name = name + "_gengo_genrule",
        message = "Generating go bindings for " + lib,
        visibility = visibility,
        testonly = testonly,
        tags = tags,
        srcs = [gx_library],
        outs = [go_file],
        tools = [gx_binder],
        cmd = (
            "$(location " + gx_binder + ")" +
            " --target_folder=$(GENDIR)"
        ),
    )
    go_library(
        name = name,
        srcs = [go_file],
        testonly = testonly,
        tags = tags,
        deps = [
            lib + "_gx",
            "//third_party/golang/errors",
            "//third_party/gxlang/gx/api",
            "//third_party/gxlang/backend/platform",
            "//third_party/gxlang/gx/interp",
            "//third_party/gxlang/gx/api/options",
            "//third_party/gxlang/gx/api/trace",
            "//third_party/gxlang/gx/api/tracer",
            "//third_party/gxlang/gx/api/values",
            "//third_party/gxlang/gx/build/ir",
            "//third_party/gxlang/gx/golang/binder/gobindings/types",
            "//third_party/gxlang/gx/golang/binder/gobindings/core",
            "//base/go:runfiles",
            # GX standard library
            "//third_party/gxlang/gx/stdlib/bindings/go/dtype:dtype_go_gx",
            "//third_party/gxlang/gx/stdlib/bindings/go/math:math_go_gx",
            "//third_party/gxlang/gx/stdlib/bindings/go/num:num_go_gx",
            "//third_party/gxlang/gx/stdlib/bindings/go/shapes:shapes_go_gx",
        ] + deps,
        visibility = visibility,
    )

def cc_gx_library(name, lib, deps = [], tags = [], visibility = None, testonly = False):
    """Generate the C++ bindings for a GX library.

    Args:
      name: name of the build rule.
      lib: GX library to write the bindings for.
      deps: list of cc_gx_library dependencies.
      tags: forwarded to generated targets
      visibility: standard Blaze attribute, passed through.
      testonly: standard Blaze attribute, passed through.
    """
    want_suffix = "_cc_gx"
    if not name.endswith(want_suffix):
        fail("target %s does not end with %s" % (name, want_suffix))
    rootname = name[:-len(want_suffix)]
    gx_binder = name[:-len(want_suffix)] + "_binder_gx"
    gx_library = name[:-len(want_suffix)] + "_gx.go"

    cc_file = rootname + ".cc"
    header_file = rootname + ".h"
    native.genrule(
        name = rootname + "_gencc_genrule",
        message = "Generating C++ bindings for " + lib,
        visibility = visibility,
        testonly = testonly,
        tags = tags,
        srcs = [gx_library],
        outs = [cc_file, header_file],
        tools = [gx_binder],
        cmd = (
            "$(location " + gx_binder + ")" +
            " --language=cc" +
            " --target_folder=$(GENDIR)"
        ),
    )
    cc_library(
        name = name,
        srcs = [cc_file],
        hdrs = [header_file],
        testonly = testonly,
        tags = tags,
        deps = [
            lib + "_gx",
            "//third_party/absl/status:statusor",
            "//third_party/gxlang/gx/golang/binder/ccgx:cppgx",
        ] + deps,
        visibility = visibility,
    )

platform_infos = {
    "cpu": {
        "args": ["--gx_platform=cpu"],
    },
    "cuda": {
        "tags": ["requires-gpu-nvidia"],
        "args": ["--gx_platform=cuda"],
    },
    "tpu": {
        "args": ["--gx_platform=tpu"],
    },
}

def go_accelerators_test(name, srcs, deps = [], platforms = [], args = [], tags = [], timeout = None, size = None):
    """Run a Go test target on multiple accelerator platform".

    Args:
      name: name of the build rule.
      srcs: Go source files for the test.
      deps: dependencies required to compile the test.
      platforms: platforms to run the test on.
      timeout: timeout of the test.
      size: size of the test.
      args: arguments to pass to the test.
      tags: tags of the test.
    """
    for platform in platforms:
        if platform not in platform_infos:
            fail("unknown platform: '" +
                 platform +
                 "'. Known platforms are: " +
                 str(platform_infos.keys()))
        platform_info = platform_infos[platform]
        if "args" in platform_info:
            args = args + platform_info["args"]
        if "tags" in platform_info:
            tags = tags + platform_info["tags"]
        go_test(
            name = platform + "_" + name,
            srcs = srcs,
            args = args,
            tags = tags,
            deps = deps,
            timeout = timeout,
            size = size,
        )

def go_test_gen_main(name, package, srcs):
    """Generate a Go file such that Go bindings tests can be run with different backends".

    Args:
      name: name of the build rule.
      package: Go package name of the library.
      srcs: Go source files for the test.
    """
    testsmain = "//third_party/gxlang/gx/golang/tools/testsmain"
    go_files = ""
    for src in srcs:
        go_files = go_files + "$(location " + src + "),"
    out = name + ".go"
    native.genrule(
        name = "genrule." + out,
        srcs = srcs,
        outs = [out],
        tools = [testsmain],
        cmd = (
            "$(location " + testsmain + ")" +
            " --go_package_name=" + package +
            " --target_folder=$(@D)" +
            " --target_name=" + out +
            " --go_files=" + go_files
        ),
    )
