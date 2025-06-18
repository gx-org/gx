
#line 1 "cgo-builtin-prolog"
#include <stddef.h>

/* Define intgo when compiling with GCC.  */
typedef ptrdiff_t intgo;

#define GO_CGO_GOSTRING_TYPEDEF
typedef struct { const char *p; intgo n; } _GoString_;
typedef struct { char *p; intgo n; intgo c; } _GoBytes_;
_GoString_ GoString(char *p);
_GoString_ GoStringN(char *p, int l);
_GoBytes_ GoBytes(void *p, int n);
char *CString(_GoString_);
void *CBytes(_GoBytes_);
void *_CMalloc(size_t);

__attribute__ ((unused))
static size_t _GoStringLen(_GoString_ s) { return (size_t)s.n; }

__attribute__ ((unused))
static const char *_GoStringPtr(_GoString_ s) { return s.p; }

#line 40 "/usr/local/google/home/degris/Temp/gx/golang/binder/cgx/cgx.go"

#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

#include "cgx.h"

// cgx_device_get_result is the return value for cgx_device_get().
struct cgx_device_get_result {
	cgx_device device;
	cgx_error error;
};

// cgx_list_functions_result is the return value when listing functions of a GX element.
struct cgx_list_functions_result {
	cgx_function* funcs;
	int num_functions;
	cgx_error error;
};

// cgx_function_signature_element describes a function parameter or return value.
struct cgx_function_signature_element {
	const char* name;
	enum cgx_value_kind kind;
};

// cgx_function_signature_result is the return value for cgx_function_signature().
struct cgx_function_signature_result {
	struct cgx_function_signature_element* parameter;
	uint32_t parameter_size;
	struct cgx_function_signature_element* result;
	uint32_t result_size;

	cgx_error error;
};

// cgx_interface_find_result is the return value for cgx_interface_find().
struct cgx_interface_find_result {
	cgx_interface iface;
	cgx_error error;
};

// cgx_function_find_result is the return value for cgx_function_find().
struct cgx_function_find_result {
	cgx_function function;
	cgx_error error;
};

// cgx_function_run_result is the return value for cgx_function_run().
struct cgx_function_run_result {
	cgx_value* values;
	uint32_t value_size;
	cgx_error error;
};

// cgx_value_new_result is the return value for cgx_value_new_*().
struct cgx_value_new_result {
	cgx_value value;
	cgx_error error;
};

// cgx_value_host_buffer_result is the return value for cgx_value_host_buffer().
struct cgx_value_host_buffer_result {
	cgx_host_buffer buffer;
	cgx_error error;
};

// cgx_value_get_struct_result is the return value for cgx_value_get_struct.
struct cgx_value_get_struct_result {
	cgx_struct strct;
	cgx_error error;
};

// cgx_struct_field_element describes a structure field.
struct cgx_struct_field_element {
	const char* name;
	enum cgx_value_kind kind;
};

// cgx_struct_field_list_result is the return value for cgx_struct_field_list().
struct cgx_struct_field_list_result {
	struct cgx_struct_field_element* field;
	uint32_t field_size;

	cgx_error error;
};

// cgx_shape_axes_result is the return value for cgx_shape_axes().
struct cgx_shape_axes_result {
	const int64_t* axis_lengths;
	uint32_t num_axes;
	cgx_error error;
};

#line 1 "cgo-generated-wrapper"


#line 1 "cgo-gcc-prolog"
/*
  If x and y are not equal, the type will be invalid
  (have a negative array count) and an inscrutable error will come
  out of the compiler and hopefully mention "name".
*/
#define __cgo_compile_assert_eq(x, y, name) typedef char name[(x-y)*(x-y)*-2UL+1UL];

/* Check at compile time that the sizes we use match our expectations. */
#define __cgo_size_assert(t, n) __cgo_compile_assert_eq(sizeof(t), (size_t)n, _cgo_sizeof_##t##_is_not_##n)

__cgo_size_assert(char, 1)
__cgo_size_assert(short, 2)
__cgo_size_assert(int, 4)
typedef long long __cgo_long_long;
__cgo_size_assert(__cgo_long_long, 8)
__cgo_size_assert(float, 4)
__cgo_size_assert(double, 8)

extern char* _cgo_topofstack(void);

/*
  We use packed structs, but they are always aligned.
  The pragmas and address-of-packed-member are only recognized as warning
  groups in clang 4.0+, so ignore unknown pragmas first.
*/
#pragma GCC diagnostic ignored "-Wunknown-pragmas"
#pragma GCC diagnostic ignored "-Wpragmas"
#pragma GCC diagnostic ignored "-Waddress-of-packed-member"
#pragma GCC diagnostic ignored "-Wunknown-warning-option"
#pragma GCC diagnostic ignored "-Wunaligned-access"

#include <errno.h>
#include <string.h>


#define CGO_NO_SANITIZE_THREAD
#define _cgo_tsan_acquire()
#define _cgo_tsan_release()


#define _cgo_msan_write(addr, sz)

CGO_NO_SANITIZE_THREAD
void
_cgo_8ae3e6ac5214_Cfunc_free(void *v)
{
	struct {
		void* p0;
	} __attribute__((__packed__, __gcc_struct__)) *_cgo_a = v;
	_cgo_tsan_acquire();
	free(_cgo_a->p0);
	_cgo_tsan_release();
}

