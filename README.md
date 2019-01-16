txtmanip
========

A tool for interactive text manipulation.

## Description

txtmanip is a text manipulation tool that possible to build you expect result rapidly and reduce the number of cycle of try and error.

## Demo


## Usage

You can select a way of start interaction from following.

- Open file with specified by the argument

```
textmanip [option] /path/to/file
```

- Receive at stdin for another command's output

```
command | textmanip [option]
```


## Configuration

The default configuration file is named `textmanip.toml`.

### enable_commands

You can only invoke commands which `enable_commands` contains in the interactive console.
When the command you want to invoke is not contained in `enable_commands`, add that command to `enable_commands`.

```
enable_commands = [
    "sed",
    "cut",
    "awk"
]
```

## License

[MIT](https://github.com/shiimaxx/txtmanip/blob/master/LICENSE)


## Author

[shiimaxx](https://github.com/shiimaxx)
