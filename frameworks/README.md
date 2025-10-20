# Framework Compatibility Mappings

## Overview

This package contains framework compatibility mappings extracted from NuGet.Client `DefaultFrameworkMappings.cs`.

These mappings define which Target Framework Monikers (TFMs) are compatible with each other and which .NET Standard versions are supported by each framework.

**Verified against:** NuGet.Client `DefaultFrameworkMappings.cs` and `CompatibilityTests.cs`

## Supported Frameworks

### Core .NET Frameworks
- .NET Framework (4.5 through 4.8.1)
- .NET Standard (1.0 through 2.1)
- .NET Core App (1.0, 1.1, 2.0, 3.0)

### Platform-Specific Frameworks
- Tizen (3.0, 4.0, 6.0)
- UAP - Universal Windows Platform (10.0.15064, wildcard)
- Windows Phone (8.0, 8.1)
- Windows Phone App (8.1)

### Xamarin Frameworks
- MonoAndroid, MonoMac, MonoTouch
- Xamarin.iOS, Xamarin.Mac
- Xamarin.TVOS, Xamarin.WatchOS
- Xamarin.PlayStation3, Xamarin.PlayStation4, Xamarin.PlayStationVita
- Xamarin.Xbox360, Xamarin.XboxOne

### Legacy Frameworks
- NetCore (4.5, 4.5.1, 5.0)
- DNXCore

## Key Mappings

### .NET Standard → .NET Framework

| .NET Standard | Min .NET Framework |
|---------------|-------------------|
| 1.0-1.1       | 4.5               |
| 1.2           | 4.5.1             |
| 1.3           | 4.6               |
| 1.4-2.0       | 4.6.1             |
| 2.1           | NOT COMPATIBLE    |

### .NET Standard → .NET Core

| .NET Standard | Min .NET Core |
|---------------|---------------|
| 1.0-1.6       | 1.0           |
| 1.7           | 1.1           |
| 2.0           | 2.0           |
| 2.1           | 3.0           |

## Critical Rules

- .NET Standard 2.1 is **NOT** compatible with any .NET Framework version
- All Xamarin mobile frameworks (iOS, Mac, Android, TVOS, WatchOS) support .NET Standard 2.1
- Xamarin console frameworks (PlayStation, Xbox) support .NET Standard 2.0
