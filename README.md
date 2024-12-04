# s0cmd README.md

# s0cmd

s0cmd is a command-line tool for downloading objects from S3-compatible APIs with high speed. It is designed to efficiently download machine learning models and other large files from object storage.

## Features

- Fast downloading of objects using parallelism.
- Support for S3-compatible APIs.
- Generates presigned URLs for secure access.
- Provides detailed logging of download progress.

## Installation

To install s0cmd, clone the repository and build the project:

```bash
git clone https://github.com/yourusername/s0cmd.git
cd s0cmd
go build -o s0cmd main.go
```

## Usage

To use s0cmd, run the following command:

```bash
./s0cmd get s3://path-to-object/xx.tensors
```

### Flags

- `-p`, `--parallelism`: Specify the number of parallel downloads.
- `-o`, `--output`: Specify the output file name.

## Examples

Download an object with default settings:

```bash
./s0cmd get s3://path-to-object/xx.tensors
```

Download an object with specified parallelism:

```bash
./s0cmd get s3://path-to-object/xx.tensors -p 4
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## License

This project is licensed under the MIT License. See the LICENSE file for more details.