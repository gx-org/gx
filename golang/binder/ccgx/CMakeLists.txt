cmake_minimum_required (VERSION 3.24)
project (ccgx)

include_directories (../../..)

add_library (ccgx STATIC cppgx.cc)
