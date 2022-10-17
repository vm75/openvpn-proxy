cmake_minimum_required(VERSION 3.20)  # CMake version check

project(openvpn-proxy)

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
