module jarvice.io

go 1.15

require (
    jarvice.io/jarvice-hpc/core v0.0.0
    jarvice.io/jarvice-hpc/logger v0.0.0
    github.com/jessevdk/go-flags v1.4.0
)

replace (
    jarvice.io/jarvice-hpc/core => ./core
    jarvice.io/jarvice-hpc/logger => ./logger
)
