export type ApiSpec = {
  name: string;
  generatedAt: string;
  sourcePath: string;
  summary: {
    endpointCount: number;
    functionCount: number;
    structCount: number;
    enumCount: number;
  };
  endpoints: Endpoint[];
  functions: ApiFunction[];
  structs: ApiStruct[];
  enums: ApiEnum[];
  warnings?: string[];
};

export type Endpoint = {
  id: string;
  name: string;
  method: string;
  path: string;
  description: string;
  signature: string;
  returnType: string;
  file: string;
  line: number;
  params: Parameter[];
  responses: ResponseDoc[];
  source?: "docblock" | "heuristic" | "config";
  deprecated?: boolean;
  example?: string;
};

export type ApiFunction = {
  id: string;
  name: string;
  description: string;
  signature: string;
  returnType: string;
  file: string;
  line: number;
  params: Parameter[];
  deprecated?: boolean;
  example?: string;
};

export type Parameter = {
  name: string;
  type: string;
  description?: string;
  direction?: string;
};

export type ResponseDoc = {
  status: string;
  type: string;
  description?: string;
};

export type ApiStruct = {
  id: string;
  name: string;
  description: string;
  file: string;
  line: number;
  fields: {
    name: string;
    type: string;
    description?: string;
  }[];
};

export type ApiEnum = {
  id: string;
  name: string;
  description: string;
  file: string;
  line: number;
  values: {
    name: string;
    description?: string;
  }[];
};

export type RuntimeConfig = {
  defaultTargetBaseUrl: string;
};

export type TryApiResponse = {
  requestedUrl: string;
  method: string;
  status: number;
  statusText: string;
  headers: Record<string, string[]>;
  body: string;
};
