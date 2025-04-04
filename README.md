# N64 Emulator Save Converter

This software is only to be used for converting Project64 save files to a compatible format for use with [Gopher64](https://github.com/gopher64/gopher64).

## Download:

[Windows x64](https://github.com/lualona66/save-converter/releases/latest/download/save-converter-windows-amd64.zip)

[Linux x64](https://github.com/lualona66/save-converter/releases/latest/download/save-converter-linux-amd64.tar.gz)


## Usage:

- 1: Running save-converter.exe will present a file picker dialog box
- 2: Pick your PJ64 save file  _( Supported save file formats: '.eep' '.mpk' '.sra' '.fla' )_
- 3: Pick the relevant N64 rom  _( Supported rom formats: '.z64' '.n64' '.v64' )_
- 4: The new converted save will be created in the same directory as the executable

## CLI Usage:

save-converter.exe <save_file> <N64_rom_file>

This will create a new file in the current directory with the converted save file.



## Gopher64 save file naming convention. ROMNAME-SHA256.extension

* **ROMNAME**: Name taken from rom header. Special characters and trailing spaces removed
* **SHA256**: Hash of rom file
* **.extension**: Save file format


## Gopher64 save location

Easily accessed by clicking "Open Saves Folder" from Gopher64's gui


* **Windows**: %USERPROFILE%\appdata\Roaming\gopher64\data\saves\
 
* **Linux**: ~/.local/share/gopher64/saves/
* **Flatpak**: ~/.var/app/io.github.gopher64.gopher64/data/gopher64/saves/

**If using Gopher64 with portable.txt then save files will be located in /portable_data/data/saves/ next to your Gopher64 executable.**






