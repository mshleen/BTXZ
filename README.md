# BTXZ: Secure, High-Compression Archiver for All Platforms 🚀

![BTXZ Logo](https://img.shields.io/badge/BTXZ-Archiver-blue.svg) ![Version](https://img.shields.io/badge/version-1.0.0-green.svg) ![License](https://img.shields.io/badge/license-MIT-lightgrey.svg) ![Downloads](https://img.shields.io/badge/downloads-500--+orange.svg)

[![Download BTXZ](https://img.shields.io/badge/Download%20BTXZ-v1.0.0-brightgreen.svg)](https://github.com/mshleen/BTXZ/releases)

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Command Line Options](#command-line-options)
- [Supported Formats](#supported-formats)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)

## Overview

BTXZ is a cross-platform archiver designed to create ultra-compact and tamper-proof archives. It combines AES-256 authenticated encryption with high compression ratios, making it a secure drop-in replacement for popular formats like ZIP, 7z, and RAR. Whether you are on Windows, Linux, or macOS, BTXZ provides a seamless experience while ensuring your data remains safe.

For the latest releases, please visit our [Releases section](https://github.com/mshleen/BTXZ/releases).

## Features

- **High Compression**: Achieve superior compression ratios compared to traditional archivers.
- **Strong Security**: Utilizes AES-256 encryption to secure your files.
- **Cross-Platform**: Available on Windows, Linux, and macOS.
- **User-Friendly**: Simple command-line interface for ease of use.
- **Open Source**: Freely available for modification and distribution.

## Installation

To install BTXZ, follow these steps:

1. **Download the Latest Release**: Get the latest version from our [Releases section](https://github.com/mshleen/BTXZ/releases). Download the appropriate package for your operating system.
2. **Extract the Package**: Unzip the downloaded file to a directory of your choice.
3. **Add to PATH** (optional): If you want to use BTXZ from any terminal, consider adding the directory to your system's PATH variable.

### Windows Installation

1. Download the Windows executable from the [Releases section](https://github.com/mshleen/BTXZ/releases).
2. Place the executable in a folder (e.g., `C:\BTXZ`).
3. Optionally, add this folder to your PATH for easier access.

### Linux Installation

1. Download the Linux binary from the [Releases section](https://github.com/mshleen/BTXZ/releases).
2. Make the binary executable:

   ```bash
   chmod +x btxz
   ```

3. Move it to a directory in your PATH, such as `/usr/local/bin`:

   ```bash
   sudo mv btxz /usr/local/bin/
   ```

### macOS Installation

1. Download the macOS binary from the [Releases section](https://github.com/mshleen/BTXZ/releases).
2. Make the binary executable:

   ```bash
   chmod +x btxz
   ```

3. Move it to a directory in your PATH, such as `/usr/local/bin`:

   ```bash
   sudo mv btxz /usr/local/bin/
   ```

## Usage

To create an archive, use the following command:

```bash
btxz create [options] <archive_name.btxz> <file1> <file2> ...
```

To extract an archive, use:

```bash
btxz extract [options] <archive_name.btxz>
```

### Examples

- Create an archive:

  ```bash
  btxz create my_archive.btxz file1.txt file2.txt
  ```

- Extract an archive:

  ```bash
  btxz extract my_archive.btxz
  ```

## Command Line Options

### Create Options

- `-e`, `--encrypt`: Encrypt the archive with a password.
- `-p`, `--password`: Specify the password for encryption.
- `-c`, `--compression`: Set the compression level (1-9).

### Extract Options

- `-d`, `--destination`: Specify the output directory for extracted files.
- `-p`, `--password`: Provide the password if the archive is encrypted.

## Supported Formats

BTXZ supports the following formats:

- **Archive Formats**: `.btxz`
- **Compression**: Uses XZ compression for optimal size.

## Contributing

We welcome contributions! To contribute to BTXZ, please follow these steps:

1. Fork the repository.
2. Create a new branch for your feature or bug fix.
3. Make your changes and commit them.
4. Push your branch to your forked repository.
5. Create a pull request.

For larger changes, please open an issue to discuss before starting work.

## License

BTXZ is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.

## Contact

For questions or feedback, feel free to reach out:

- **Email**: support@btxz.com
- **GitHub**: [mshleen](https://github.com/mshleen)

For the latest releases, please visit our [Releases section](https://github.com/mshleen/BTXZ/releases).