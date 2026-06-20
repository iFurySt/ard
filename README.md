# ard

中文版本：[`ard-cn`](https://github.com/iFurySt/ard-cn)

## Intro

An agent-first base repo template for building any product you want. For a closer look at how this approach works in daily practice, see [Daily Harness](https://www.ifuryst.com/en/blog/2026/daily-harness/).

## Quick Start

Use GitHub's template flow from the top right of this repository:

1. Select **Use this template**.
2. Select [**Create a new repository**](https://github.com/new?template_name=ard&template_owner=iFurySt).

Or initialize a new or existing repository with [`harness-cli`](https://github.com/iFurySt/harness-cli).
Install it from npm first:

```sh
npm install -g @ifuryst/harness-cli
```

Then run:

```sh
harness-cli init --language en
```

`harness-cli` requires Node.js 18+ and Go on your `PATH`.

## License

[Apache-2.0](LICENSE)

## Note

This approach comes from our own exploration, while also drawing on some ideas from OpenAI's [harness engineering write-up](https://openai.com/index/harness-engineering/).
