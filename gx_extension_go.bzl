# Copyright 2026 Google LLC
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

"""Provides gx_extension_go to import a Go package into GX"""

load("//tools/build_defs/go:go_library.bzl", "go_library")

def _is_go_library_kind_impl(target, ctx):
    if ctx.rule.kind != "go_library":
        return fail("target {} has unexpected type {}: expect go_library".format(target.label, ctx.rule.kind))
    return []

_is_go_library_kind_aspect = aspect(
    implementation = _is_go_library_kind_impl,
    attr_aspects = ["lib"],
)

GxExtensionGoInfo = provider(
    doc = "Provides the generated GX file from _gx_extension_go_gen_rule.",
    fields = {
        "go_file": "The generated .go file",
        "gx_file": "The generated .gx file",
    },
)

def _generate_all(ctx):
    lib_name = ctx.label.name[:-4]
    out_go_file = ctx.actions.declare_file(lib_name + ".go")
    out_gx_file = ctx.actions.declare_file(lib_name + ".gx")
    lib_files = ctx.attr.lib[DefaultInfo].files.to_list()
    args = ctx.actions.args()
    args.add("importgo")
    args.add("--source_filename=" + lib_name)
    args.add(lib_files[1])
    ctx.actions.run(
        outputs = [out_gx_file, out_go_file],
        inputs = [lib_files[1]],
        executable = ctx.executable._tool,
        arguments = [args],
        mnemonic = "gximportgo",
    )
    return [
        GxExtensionGoInfo(
            go_file = out_go_file,
            gx_file = out_gx_file,
        ),
    ]

_generate_all_rule = rule(
    implementation = _generate_all,
    attrs = {
        "lib": attr.label(
            mandatory = True,
            doc = "Go package to import in GX",
            aspects = [_is_go_library_kind_aspect],
        ),
        "_tool": attr.label(
            executable = True,
            cfg = "exec",
            default = Label("//third_party/gxlang/gx/gx:gx"),
        ),
    },
)

def _go_srcs(ctx):
    src = ctx.attr.gen[GxExtensionGoInfo].go_file
    return [DefaultInfo(files = depset([src]))]

_go_srcs_rule = rule(
    implementation = _go_srcs,
    attrs = {
        "gen": attr.label(
            mandatory = True,
            providers = [GxExtensionGoInfo],
        ),
    },
)

def _gx_srcs(ctx):
    src = ctx.attr.gen[GxExtensionGoInfo].gx_file
    return [DefaultInfo(files = depset([src]))]

_gx_srcs_rule = rule(
    implementation = _gx_srcs,
    attrs = {
        "gen": attr.label(
            mandatory = True,
            providers = [GxExtensionGoInfo],
        ),
    },
)

def gx_extension_go(name, lib, **kwargs):
    """Import a Go package into GX.

    Args:
      name: name of the build rule.
      lib: Go package to import in GX.
      **kwargs: forwarded to generated go_library.
    """
    all_srcs_name = name + "_gen"
    _generate_all_rule(
        name = all_srcs_name,
        lib = lib,
    )
    go_srcs_name = name + "_go_srcs"
    _go_srcs_rule(
        name = go_srcs_name,
        gen = ":" + all_srcs_name,
    )
    gx_srcs_name = name + "_gx_srcs"
    _gx_srcs_rule(
        name = gx_srcs_name,
        gen = ":" + all_srcs_name,
    )
    go_library(
        name = name,
        srcs = [":" + go_srcs_name],
        embedsrcs = [":" + gx_srcs_name],
        deps = [
            lib,
            "//third_party/gxlang/gx/api/values",
            "//third_party/gxlang/gx/build/importers/embedpkg",
            "//third_party/gxlang/gx/build/importers",
            "//third_party/gxlang/gx/build/ir",
            "//third_party/gxlang/gx/interp/engine",
            "//third_party/gxlang/gx/stdlib/builtin",
        ],
        **kwargs
    )
