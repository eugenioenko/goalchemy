// std.errors.new: errors.New returns a distinct *errors.errorString.
import { box, typeDesc, type Box } from "../types/iface.ts";

interface ErrorString {
  s: string;
}

export const ERRORS_ERROR_STRING = typeDesc({
  name: "*errors.errorString",
  kind: "pointer",
  eq: (a: ErrorString, b: ErrorString) => a === b,
  key: (a: ErrorString) => a,
  methods: { Error: (p: ErrorString) => p.s },
});

export function stdErrorsNew(text: string): Box {
  return box(ERRORS_ERROR_STRING, { s: text });
}
