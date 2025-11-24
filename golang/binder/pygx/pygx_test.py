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

import numpy as np

from google3.testing.pybase import googletest
from google3.third_party.gxlang.gx.golang.binder.pygx import pygx
from google3.third_party.gxlang.gx.golang.binder.pygx.testing import pygxtesting


class PygxTest(googletest.TestCase):
  BASIC_PACKAGE = "github.com/gx-org/gx/tests/bindings/basic"
  PARAMETERS_PACKAGE = (
      "github.com/gx-org/gx/tests/bindings/parameters"
  )
  PKGVARS_PACKAGE = (
      "github.com/gx-org/gx/tests/bindings/pkgvars"
  )
  UNEXPORTED_PACKAGE = (
      "github.com/gx-org/gx/tests/bindings/unexported"
  )

  def testGetDevice(self):
    runtime = pygxtesting.new_runtime()
    runtime.get_device(0)

  def testLoadPackage(self):
    runtime = pygxtesting.new_runtime()
    pkg = runtime.load(self.BASIC_PACKAGE)
    self.assertEqual(pkg.name(), "basic")
    self.assertEqual(pkg.fullname(), self.BASIC_PACKAGE)

  def testBuildPackage_NotFound(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    with self.assertRaises(RuntimeError):
      device.build_package("gx/fake")

  def testPackage_ListInterfaces(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    ifaces = package.list_interfaces()
    wants = [
        "Empty",
        "Basic",
    ]
    self.assertLen(ifaces, len(wants))
    for i, vr in enumerate(ifaces):
      self.assertEqual(vr.name(), wants[i])

  def testPackage_ListStaticVars(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.PKGVARS_PACKAGE)
    staticvars = package.list_static_vars()
    wants = [
        "Var1",
        "Size",
    ]
    self.assertLen(staticvars, len(wants))
    for i, vr in enumerate(staticvars):
      self.assertEqual(vr.name(), wants[i])

  def testFindStaticVar(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.PKGVARS_PACKAGE)
    self.assertTrue(package.has_static_var("Var1"))
    static_var = package.find_static_var("Var1")
    self.assertEqual(static_var.name(), "Var1")

  def testFindStaticVar_NotFound(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.PKGVARS_PACKAGE)
    self.assertFalse(package.has_static_var("Fake"))
    with self.assertRaises(RuntimeError):
      package.find_static_var("Fake")

  def testSetStaticVar(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.PKGVARS_PACKAGE)
    static_var = package.find_static_var("Var1")
    static_var.set_value(42)
    function = package.find_function("ReturnVar1")
    result = function.run([])
    values = result.return_values()
    self.assertEqual(values[0].int32_value(), 42)

  def testPackage_ListFunctions(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    funcs = package.list_functions()
    wants = [
        "ReturnFloat32",
        "ReturnArrayFloat32",
        "ReturnMultiple",
        "New",
        "AddPrivate",
    ]
    self.assertLen(funcs, len(wants))
    for i, fn in enumerate(funcs):
      self.assertEqual(fn.name(), wants[i])

  def testFindFunction(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    self.assertTrue(package.has_function("ReturnArrayFloat32"))
    function = package.find_function("ReturnArrayFloat32")
    self.assertEqual(function.name(), "ReturnArrayFloat32")
    self.assertEqual(
        function.doc(), "ReturnArrayFloat32 returns a float32 tensor.\n"
    )
    self.assertEqual(
        str(function),
        "func ReturnArrayFloat32() [2]float32 {\n\treturn [2]float32{4.2,"
        " 42}\n}",
    )
    self.assertEqual(function.num_params(), 0)

  def testFindFunction_Signature(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.PARAMETERS_PACKAGE)
    function = package.find_function("AddInt")
    signature = function.signature()
    self.assertLen(signature.parameters(), 2)
    self.assertEqual(signature.parameters()[0].name, "x")
    self.assertEqual(signature.parameters()[0].kind, pygx.Kind.CGX_INT64)
    self.assertEqual(signature.parameters()[1].name, "y")
    self.assertEqual(signature.parameters()[1].kind, pygx.Kind.CGX_INT64)
    self.assertLen(signature.results(), 1)
    self.assertEqual(signature.results()[0].name, "")
    self.assertEqual(signature.results()[0].kind, pygx.Kind.CGX_INT64)

  def testFindFunction_NotFound(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    self.assertFalse(package.has_function("Fake"))
    with self.assertRaises(RuntimeError):
      package.find_function("Fake")

  def testFindInterface_NotFound(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    with self.assertRaises(RuntimeError):
      package.find_interface("Complicated")

  def testRunFunction_ReturnFloat32(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    function = package.find_function("ReturnFloat32")
    result = function.run([])

    values = result.return_values()
    self.assertLen(values, 1)
    self.assertEqual(values[0].kind(), pygx.Kind.CGX_FLOAT32)
    self.assertAlmostEqual(values[0].float32_value(), 4.2, places=6)

  def testRunFunction_ReturnArrayFloat32(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    function = package.find_function("ReturnArrayFloat32")
    result = function.run([])

    values = result.return_values()
    self.assertLen(values, 1)
    self.assertEqual(values[0].kind(), pygx.Kind.CGX_ARRAY)
    self.assertIsNone(values[0].interface_type(package))
    self.assertEqual(
        str(values[0]),
        "DeviceArray{DeviceID: 0}: [2]float32{4.2, 42}",
    )

    shape = values[0].shape()
    self.assertEqual(shape.size(), 2)
    self.assertSequenceEqual(shape.axes(), [2])

    array = values[0].to_ndarray()
    np.testing.assert_array_equal(array, np.array([4.2, 42], dtype=np.float32))

  def testRunFunction_ReturnMultiple(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    function = package.find_function("ReturnMultiple")
    result = function.run([])

    values = result.return_values()
    self.assertLen(values, 3)
    self.assertEqual(values[0].kind(), pygx.Kind.CGX_INT32)
    self.assertAlmostEqual(values[0].int32_value(), 0, places=6)
    self.assertEqual(values[1].kind(), pygx.Kind.CGX_FLOAT32)
    self.assertAlmostEqual(values[1].float32_value(), 1, places=6)
    self.assertEqual(values[2].kind(), pygx.Kind.CGX_FLOAT64)
    self.assertAlmostEqual(values[2].float64_value(), 2.71828, places=6)

  def testRunFunction_AddInt(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.PARAMETERS_PACKAGE)
    function = package.find_function("AddInt")
    self.assertEqual(function.num_params(), 2)
    self.assertEqual(function.param_dtype(0), pygx.Kind.CGX_INT64)
    self.assertEqual(function.param_dtype(1), pygx.Kind.CGX_INT64)
    result = function.run(
        [
            pygx.Value.from_int64(device, 100),
            pygx.Value.from_int64(device, 200),
        ],
    )

    values = result.return_values()
    self.assertLen(values, 1)
    self.assertEqual(values[0].kind(), pygx.Kind.CGX_INT64)
    self.assertEqual(values[0].int64_value(), 300)

  def testRunFunction_MissingParameters(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.PARAMETERS_PACKAGE)
    function = package.find_function("AddInt")
    with self.assertRaises(RuntimeError):
      function.run([])

  def testNewShape_Float32Array(self):
    shape = pygx.Shape.new(pygx.Kind.CGX_FLOAT32, (3, 4))
    self.assertEqual(shape.size(), 3 * 4)
    self.assertSequenceEqual(shape.axes(), (3, 4))

  def testNewValue_FromNdarray(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    data = np.array([4.2, 42], dtype=np.float32)
    value = pygx.Value.from_ndarray(device, data)

    shape = value.shape()
    self.assertEqual(shape.size(), 2)
    self.assertSequenceEqual(shape.axes(), [2])

    np.testing.assert_array_equal(value.to_ndarray(), data)

  def testRunMethod_AddPrivate(self):
    device = pygxtesting.new_runtime().get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    new_func = package.find_function("New")
    new_result = new_func.run([]).return_values()
    basic_interface = new_result[0].interface_type(package)
    self.assertIsNotNone(basic_interface)
    self.assertEqual(self.BASIC_PACKAGE, basic_interface.package_name())
    self.assertEqual("Basic", basic_interface.name())
    add_func = basic_interface.find_method("AddPrivate")
    add_result = add_func.run_method(new_result[0], []).return_values()
    self.assertEqual(add_result[0].int32_value(), 6)

  def testRunMethod_FindInterface(self):
    device = pygxtesting.new_runtime().get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    basic_interface = package.find_interface("Basic")
    new_func = package.find_function("New")
    new_result = new_func.run([]).return_values()
    self.assertIsNotNone(basic_interface)
    add_func = basic_interface.find_method("AddPrivate")
    add_result = add_func.run_method(new_result[0], []).return_values()
    self.assertEqual(add_result[0].int32_value(), 6)

  def testInterface_ListMethods(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    basic = package.find_interface("Basic")
    funcs = basic.list_methods()
    wants = ["AddPrivate", "SetFloat"]
    self.assertLen(funcs, len(wants))
    for i, fn in enumerate(funcs):
      self.assertEqual(fn.name(), wants[i])

  def testStruct_GetField(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    new_func = package.find_function("New")
    new_result = new_func.run([]).return_values()
    struct_value = new_result[0].struct_type()
    self.assertIsNotNone(struct_value)
    value = struct_value.get_field("Int")
    self.assertEqual(value.kind(), pygx.Kind.CGX_INT32)
    self.assertEqual(value.int32_value(), 42)

  def testStruct_SetField(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    new_func = package.find_function("New")
    new_result = new_func.run([]).return_values()
    struct_value = new_result[0].struct_type()
    self.assertIsNotNone(struct_value)
    struct_value.set_field("Int", pygx.Value.from_int32(device, 24))
    value = struct_value.get_field("Int")
    self.assertEqual(value.kind(), pygx.Kind.CGX_INT32)
    self.assertEqual(value.int32_value(), 24)

  def testStruct_ListFields(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.BASIC_PACKAGE)
    new_func = package.find_function("New")
    new_result = new_func.run([]).return_values()
    struct_value = new_result[0].struct_type()
    self.assertIsNotNone(struct_value)
    fields = struct_value.list_fields()
    self.assertLen(fields, 7)
    self.assertEqual(fields[0].name, "Int")
    self.assertEqual(fields[0].kind, pygx.Kind.CGX_INT32)

  def testUnexportedStructure(self):
    runtime = pygxtesting.new_runtime()
    device = runtime.get_device(0)
    package = device.build_package(self.UNEXPORTED_PACKAGE)
    unexported_struct = package.find_function("New").run([]).return_values()[0]
    method_a = unexported_struct.interface_type(package).find_method("A")
    got = method_a.run_method(unexported_struct, []).return_values()[0]
    self.assertEqual(got.float32_value(), 42)


if __name__ == "__main__":
  googletest.main()
