import { Type } from "@earendil-works/pi-ai";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export default function (pi: ExtensionAPI) {
  pi.registerTool({
    name: "probe_lookup",
    label: "Probe Lookup",
    description: "Look up a value in the agent request packaging probe.",
    promptSnippet: "Look up a probe value by key",
    parameters: Type.Object({
      key: Type.String({ description: "Value key" }),
    }),
    async execute(_id, params) {
      return { content: [{ type: "text", text: `value:${params.key}` }], details: {} };
    },
  });

  pi.registerTool({
    name: "probe_calculate",
    label: "Probe Calculate",
    description: "Calculate a simple sum in the agent request packaging probe.",
    promptSnippet: "Calculate the sum of two numbers",
    parameters: Type.Object({
      left: Type.Number({ description: "Left operand" }),
      right: Type.Number({ description: "Right operand" }),
    }),
    async execute(_id, params) {
      return {
        content: [{ type: "text", text: String(params.left + params.right) }],
        details: {},
      };
    },
  });

  pi.registerTool({
    name: "probe_notify",
    label: "Probe Notify",
    description: "Record a notification in the agent request packaging probe.",
    promptSnippet: "Record a probe notification",
    parameters: Type.Object({
      message: Type.String({ description: "Notification text" }),
      severity: Type.Union([
        Type.Literal("info"),
        Type.Literal("warning"),
        Type.Literal("error"),
      ]),
    }),
    async execute(_id, params) {
      return {
        content: [{ type: "text", text: `${params.severity}:${params.message}` }],
        details: {},
      };
    },
  });
}
