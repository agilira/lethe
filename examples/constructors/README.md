# Constructor Examples

This directory contains examples demonstrating all the different ways to create Lethe Logger instances.

## Constructor Functions Demonstrated

- **`New()`** - Legacy constructor (backward compatible)
- **`NewSimple()`** - Modern string-based constructor  
- **`NewWithDefaults()`** - Production defaults
- **`NewDaily()`** - Daily rotation setup
- **`NewWeekly()`** - Weekly rotation setup
- **`NewDevelopment()`** - Development-optimized setup
- **`NewWithConfig()`** - Full configuration control

## Running the Examples

```bash
go run constructor_examples.go
```

This will create example log files in the `./logs/` directory and demonstrate each constructor function with practical use cases.

## 📋 What You'll Learn

- How to choose the right constructor for your use case
- Configuration differences between production and development
- String-based configuration with MaxAgeStr and MaxSizeStr
- Error handling and validation
- Integration with Go's standard library

## Recommended Usage

- **Production Apps**: Use `NewWithDefaults()`
- **Development**: Use `NewDevelopment()`  
- **Time-based Rotation**: Use `NewDaily()` or `NewWeekly()`
- **Full Control**: Use `NewWithConfig()`
- **Legacy Code**: Use `New()` for drop-in compatibility
