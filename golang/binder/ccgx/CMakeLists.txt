cmake_minimum_required (VERSION 3.24)
project (ccgx)

include_directories (${CMAKE_CURRENT_LIST_DIR}/../../..)

add_library (ccgx STATIC ${CMAKE_CURRENT_LIST_DIR}/cppgx.cc)
