cmake_minimum_required(VERSION 3.0.0)
project(openvpn-proxy VERSION 0.1.0)

include(CTest)
enable_testing()

set(CMAKE_CXX_STANDARD 20)            # Enable c++20 standard

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

set(CMAKE_CXX_FLAGS "-Wall -Wextra -fno-rtti -fcoroutines -fno-exceptions -fmodules-ts")
set(CMAKE_CXX_FLAGS_DEBUG "-g")
set(CMAKE_CXX_FLAGS_RELEASE "-O3 -s -g0 -fvisibility=hidden -fvisibility-inlines-hidden -fno-unwind-tables -fno-omit-frame-pointer")

add_executable(${PROJECT_NAME}
    main.cpp
)

set(CPACK_PROJECT_NAME ${PROJECT_NAME})
set(CPACK_PROJECT_VERSION ${PROJECT_VERSION})
include(CPack)
