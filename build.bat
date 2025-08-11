@echo off
echo Building timeline2svg...
go build -o timeline2svg.exe main.go
if %errorlevel% equ 0 (
    echo Build successful! Executable: timeline2svg.exe
    echo.
    echo Usage: timeline2svg.exe ^<csv_file^> [config_file] [output_file]
    echo Example: timeline2svg.exe sample-data.csv
) else (
    echo Build failed!
)
