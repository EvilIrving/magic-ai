# Pi 中转站调研结论

这份说明只记录已经在当前仓库源码和探测结果里确认过的事实，目的是避免以后只看到 `models.json` 或探测脚本时，不知道这些配置到底验证到什么程度。

认证顺序是先读环境变量，再读用户的 shell 启动文件，最后回落到 `~/.pi/agent/auth.json`。对应的环境变量分别是 `SHU_API_KEY`、`SHU_DEEPSEEK_API_KEY` 和 `SHU_GROK_API_KEY`，三个 provider 都映射到同一个网关地址 `https://api.shu.cool/v1`。

当前 `models.json` 里这三组 provider 都使用 `openai-responses` 作为 API 类型，没有看到单独的模型级 `api` 覆盖项。这意味着 Pi 这边的主配置面是按 OpenAI Responses 兼容层来组织的，而不是靠模型各自切换成不同协议。

从探测脚本和返回体来看，`reasoning.effort` 或 `reasoning_effort` 才是实际会被请求使用的强度字段，`mode` 则是返回体里出现的状态字段，不是当前这份配置里可直接改写的本地开关。现有证据里看到的 `reasoning.mode` 是 `standard`，`service_tier` 则是网关回传的 `auto` 或 `default`，没有证据表明只改 `models.json` 就能把它变成 OpenAI 的 `priority` 快速模式。

`models.json` 里写的 `contextWindow` 和 `maxTokens` 更像是本地目录信息，不是网关返回出来的强校验结果。现有短请求只证明了网关能接受多种长度字段，并不能证明这些上限已经被真实模型严格执行。

`responses`、`chat_completions` 和 `messages` 三条路径都已经在探测脚本里被单独测过，但不同模型对参数的接受度并不一致。尤其是思考强度、严格工具调用、结构化输出和长度边界，不能只按状态码一眼判断全部兼容。

目前比较稳妥的结论是，Pi 中转站已经能稳定承载 OpenAI Responses 兼容层的主路径，认证来源也已经确认清楚，但 `priority` 这种“快速模式”是否能生效，要看中转站和上游是否显式支持对应请求字段，而不是简单改本地模型表就能保证。
