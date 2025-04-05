# N64 Emulator Save Converter

This software renames/converts N64 emulator save data to a compatible format for use with **gopher64**.
### Currently Supported Emulator Saves
- Ares
- Project64
- RMG
- Simple64


## Download:

[Windows x64](https://github.com/lualona66/save-converter/releases/latest/download/save-converter-windows-amd64.zip)

[Linux x64](https://github.com/lualona66/save-converter/releases/latest/download/save-converter-linux-amd64.tar.gz)


## Usage:

- 1: Running save-converter.exe will present a file picker dialog box
- 2: Pick the save file you want to copy data from
  - ( Supported save file formats: '.eep/.eeprom' '.mpk/.pak' '.sra/.ram' '.fla/.flash' )
- 3: Pick the relevant N64 ROM
  - ( Supported ROM formats: '.z64' '.n64' '.v64' )
- 4: A new save will be created in the same directory as the save-converter executable

## CLI Usage:

save-converter.exe <save_file> <N64_rom_file>

This will create a new file in the current directory with the converted save file.


## gopher64 save file naming convention:
### ROMNAME-SHA256.extension

* **ROMNAME**: Name taken from ROM header. Special characters and trailing spaces removed
* **SHA256**: Hash of ROM file
* **.extension**: Save file format


## gopher64 save location:

Easily accessible by clicking "**Open Saves Folder**" from Gopher64's interface.


* **Windows**: %USERPROFILE%\appdata\Roaming\gopher64\data\saves\

* **Linux**: ~/.local/share/gopher64/saves/
* **Flatpak**: ~/.var/app/io.github.gopher64.gopher64/data/gopher64/saves/

**If using gopher64 with portable.txt then save files will be located in /portable_data/data/saves/ next to your Gopher64 executable.**

