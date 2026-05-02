import { flexsearchFromSource } from "fumadocs-core/search/flexsearch"

import { source } from "@/lib/source"

const server = flexsearchFromSource(source)

export const { GET } = server
