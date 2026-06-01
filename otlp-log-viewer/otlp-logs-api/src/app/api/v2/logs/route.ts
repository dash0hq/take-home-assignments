import { faker } from "@faker-js/faker";
import {
	IExportLogsServiceRequest,
	IResourceLogs,
	ILogRecord,
} from "@opentelemetry/otlp-transformer/build/src/logs/internal-types";

export const runtime = "edge";

// Responses are deterministic within this window: same bucket -> same seed
// and same time bounds -> identical payload. Window rolls forward over time.
const SEED_WINDOW_MS = 10_000;

// Reference date for the active window, set per-request in generatedMockedData.
// Pins faker.date.* anchor points to the window instead of live `now` so the
// payload stays byte-identical across calls within the same window. Safe as a
// module-scoped var because generation is fully synchronous (no awaits).
let windowRefDate = new Date(0);

export async function GET() {
	return Response.json(generatedMockedData(), {
		headers: {
			"Access-Control-Allow-Origin": "*",
		},
	});
}

const severityTexts = [
	"UNSPECIFIED",
	"TRACE",
	"TRACE",
	"TRACE",
	"TRACE",
	"DEBUG",
	"DEBUG",
	"DEBUG",
	"DEBUG",
	"INFO",
	"INFO",
	"INFO",
	"INFO",
	"WARN",
	"WARN",
	"WARN",
	"WARN",
	"ERROR",
	"ERROR",
	"ERROR",
	"ERROR",
	"FATAL",
	"FATAL",
	"FATAL",
	"FATAL",
];

const httpMethods = ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"];
const httpPaths = [
	"/api/users",
	"/api/orders",
	"/api/products",
	"/api/auth/login",
	"/api/auth/refresh",
	"/api/webhooks/stripe",
	"/api/health",
	"/api/search",
	"/api/notifications",
	"/api/uploads",
];
const errorTypes = [
	"TypeError",
	"RangeError",
	"SyntaxError",
	"ReferenceError",
	"ConnectionError",
	"TimeoutError",
	"ValidationError",
	"AuthenticationError",
	"PermissionDeniedError",
	"NotFoundError",
];
const nodeModules = [
	"node:internal/process/task_queues",
	"node:internal/modules/cjs/loader",
	"node:internal/modules/run_main",
	"node:internal/main/run_main_module",
];

function generateStackTrace(): string {
	const error = faker.helpers.arrayElement(errorTypes);
	const message = faker.git.commitMessage();
	const depth = faker.number.int({ min: 4, max: 12 });

	const frames: string[] = [];
	for (let i = 0; i < depth; i++) {
		const modulePath = faker.helpers.arrayElement([
			`/app/src/${faker.helpers.arrayElement(["services", "controllers", "middleware", "handlers", "utils", "lib"])}/${faker.word.noun()}.ts`,
			`/app/node_modules/${faker.word.noun()}-${faker.word.noun()}/lib/${faker.word.noun()}.js`,
			faker.helpers.arrayElement(nodeModules),
		]);
		const funcName = faker.helpers.arrayElement([
			`${faker.word.verb()}${faker.word.noun().charAt(0).toUpperCase()}${faker.word.noun().slice(1)}`,
			`async ${faker.word.verb()}`,
			"processTicksAndRejections",
			"Object.<anonymous>",
			`${faker.word.noun()}.${faker.word.verb()}`,
		]);
		const line = faker.number.int({ min: 1, max: 500 });
		const col = faker.number.int({ min: 1, max: 80 });
		frames.push(`    at ${funcName} (${modulePath}:${line}:${col})`);
	}

	return `${error}: ${message}\n${frames.join("\n")}`;
}

function generateJsonBody(): string {
	const variant = faker.number.int({ min: 0, max: 3 });
	switch (variant) {
		case 0:
			return JSON.stringify({
				request: {
					method: faker.helpers.arrayElement(httpMethods),
					path: faker.helpers.arrayElement(httpPaths),
					headers: {
						"content-type": "application/json",
						"x-request-id": faker.string.uuid(),
						"user-agent": faker.internet.userAgent(),
					},
					body: {
						userId: faker.string.uuid(),
						action: faker.word.verb(),
					},
				},
				response: {
					status: faker.helpers.arrayElement([
						200, 201, 400, 401, 403, 404, 500, 502, 503,
					]),
					duration_ms: faker.number.int({ min: 1, max: 5000 }),
				},
			});
		case 1:
			return JSON.stringify({
				error: {
					type: faker.helpers.arrayElement(errorTypes),
					message: faker.hacker.phrase(),
					code: `ERR_${faker.string.alpha({ length: 4, casing: "upper" })}_${faker.number.int({ min: 100, max: 999 })}`,
					details: {
						userId: faker.string.uuid(),
						resource: faker.helpers.arrayElement(httpPaths),
						retryable: faker.datatype.boolean(),
						timestamp: faker.date
							.recent({ days: 1, refDate: windowRefDate })
							.toISOString(),
					},
				},
			});
		case 2:
			return JSON.stringify({
				event: "db_query",
				query: `SELECT ${faker.helpers.arrayElement(["*", "id, name, email", "COUNT(*)"])} FROM ${faker.word.noun()}s WHERE ${faker.word.noun()}_id = $1`,
				params: [faker.string.uuid()],
				duration_ms: faker.number.float({
					min: 0.1,
					max: 2000,
					fractionDigits: 2,
				}),
				rows_affected: faker.number.int({ min: 0, max: 500 }),
				connection_pool: {
					active: faker.number.int({ min: 0, max: 20 }),
					idle: faker.number.int({ min: 0, max: 10 }),
					waiting: faker.number.int({ min: 0, max: 5 }),
				},
			});
		default:
			return JSON.stringify({
				kafka: {
					topic: `${faker.word.noun()}-events`,
					partition: faker.number.int({ min: 0, max: 11 }),
					offset: faker.number.int({ min: 0, max: 999999 }),
					key: faker.string.uuid(),
				},
				payload: {
					type: faker.helpers.arrayElement([
						"created",
						"updated",
						"deleted",
						"published",
						"archived",
					]),
					entity: faker.word.noun(),
					entityId: faker.string.uuid(),
					version: faker.number.int({ min: 1, max: 20 }),
				},
				metadata: {
					producer: `${faker.word.noun()}-service`,
					timestamp: faker.date
						.recent({ days: 1, refDate: windowRefDate })
						.toISOString(),
					correlationId: faker.string.uuid(),
				},
			});
	}
}

function generateLongBody(): string {
	const context = faker.helpers.arrayElement([
		`Request processing failed after ${faker.number.int({ min: 1, max: 30 })} retries. ` +
			`Original request: ${faker.helpers.arrayElement(httpMethods)} ${faker.helpers.arrayElement(httpPaths)} ` +
			`from client ${faker.internet.ip()} (User-Agent: ${faker.internet.userAgent()}). ` +
			`Last error: ${faker.helpers.arrayElement(errorTypes)}: ${faker.hacker.phrase()}. ` +
			`Circuit breaker state: ${faker.helpers.arrayElement(["open", "half-open", "closed"])}. ` +
			`Total elapsed time: ${faker.number.int({ min: 100, max: 30000 })}ms. ` +
			`Upstream service: ${faker.word.noun()}-service (${faker.internet.ip()}:${faker.internet.port()}). ` +
			`Request ID: ${faker.string.uuid()}. Trace ID: ${faker.string.hexadecimal({ length: 32 }).slice(2)}.`,

		`Database migration ${faker.system.semver()} completed with warnings. ` +
			`Applied ${faker.number.int({ min: 1, max: 15 })} of ${faker.number.int({ min: 15, max: 30 })} pending migrations. ` +
			`Skipped migrations: ${Array.from({ length: faker.number.int({ min: 1, max: 4 }) }, () => `${faker.date.recent({ days: 30, refDate: windowRefDate }).toISOString().slice(0, 10)}_${faker.word.verb()}_${faker.word.noun()}_table`).join(", ")}. ` +
			`Total rows affected: ${faker.number.int({ min: 0, max: 100000 })}. ` +
			`Execution time: ${faker.number.float({ min: 0.5, max: 120, fractionDigits: 2 })}s. ` +
			`Connection pool: ${faker.number.int({ min: 1, max: 20 })} active, ${faker.number.int({ min: 0, max: 10 })} idle. ` +
			`Advisory lock acquired: ${faker.datatype.boolean()}. Schema version now at: ${faker.number.int({ min: 1, max: 200 })}.`,

		`Health check probe detected degraded performance on ${faker.word.noun()}-service. ` +
			`CPU utilization: ${faker.number.float({ min: 60, max: 99, fractionDigits: 1 })}% (threshold: 80%). ` +
			`Memory: ${faker.number.int({ min: 512, max: 4096 })}MB / ${faker.number.int({ min: 4096, max: 16384 })}MB. ` +
			`Active connections: ${faker.number.int({ min: 100, max: 10000 })}. ` +
			`P99 latency: ${faker.number.int({ min: 200, max: 5000 })}ms (SLO: 500ms). ` +
			`GC pause time: ${faker.number.float({ min: 10, max: 500, fractionDigits: 1 })}ms. ` +
			`Thread pool: ${faker.number.int({ min: 50, max: 200 })} active / ${faker.number.int({ min: 200, max: 500 })} max. ` +
			`Last successful deploy: ${faker.date.recent({ days: 7, refDate: windowRefDate }).toISOString()}. Pod: ${faker.word.noun()}-${faker.string.alphanumeric(8)}.`,
	]);
	return context;
}

function generateLogBody(): string {
	const roll = faker.number.float({ min: 0, max: 1 });

	if (roll < 0.35) {
		// Short: git commit message style
		return faker.git.commitMessage();
	} else if (roll < 0.55) {
		// Medium: a sentence or two
		return faker.hacker.phrase();
	} else if (roll < 0.7) {
		// Long: multi-sentence verbose log
		return generateLongBody();
	} else if (roll < 0.85) {
		// JSON structured body
		return generateJsonBody();
	} else {
		// Stack trace
		return generateStackTrace();
	}
}

function generateLogAttributes(
	severityNumber: number
): Array<{
	key: string;
	value: { stringValue?: string; intValue?: number; boolValue?: boolean };
}> {
	const attrs: Array<{
		key: string;
		value: {
			stringValue?: string;
			intValue?: number;
			boolValue?: boolean;
		};
	}> = [];

	// ~60% of logs get HTTP attributes
	if (faker.number.float({ min: 0, max: 1 }) < 0.6) {
		const method = faker.helpers.arrayElement(httpMethods);
		const path = faker.helpers.arrayElement(httpPaths);
		const statusCode =
			severityNumber >= 17
				? faker.helpers.arrayElement([500, 502, 503, 504])
				: severityNumber >= 13
					? faker.helpers.arrayElement([400, 401, 403, 404, 429])
					: faker.helpers.arrayElement([200, 201, 204]);

		attrs.push(
			{
				key: "http.method",
				value: { stringValue: method },
			},
			{
				key: "http.url",
				value: {
					stringValue: `https://${faker.word.noun()}-service.internal${path}`,
				},
			},
			{
				key: "http.status_code",
				value: { intValue: statusCode },
			},
			{
				key: "http.response_time_ms",
				value: {
					intValue: faker.number.int({ min: 1, max: 5000 }),
				},
			}
		);
	}

	// Error-level logs get error attributes
	if (severityNumber >= 17) {
		attrs.push(
			{
				key: "error.type",
				value: {
					stringValue: faker.helpers.arrayElement(errorTypes),
				},
			},
			{
				key: "error.stack_depth",
				value: {
					intValue: faker.number.int({ min: 3, max: 15 }),
				},
			}
		);
	}

	// ~40% get a thread/trace ID
	if (faker.number.float({ min: 0, max: 1 }) < 0.4) {
		attrs.push(
			{
				key: "trace.id",
				value: {
					stringValue: faker.string.hexadecimal({ length: 32 }).slice(2),
				},
			},
			{
				key: "span.id",
				value: {
					stringValue: faker.string.hexadecimal({ length: 16 }).slice(2),
				},
			}
		);
	}

	// ~30% get deployment info
	if (faker.number.float({ min: 0, max: 1 }) < 0.3) {
		attrs.push(
			{
				key: "deployment.environment",
				value: {
					stringValue: faker.helpers.arrayElement([
						"production",
						"staging",
						"development",
					]),
				},
			},
			{
				key: "host.name",
				value: {
					stringValue: `${faker.word.noun()}-${faker.string.alphanumeric(6)}`,
				},
			}
		);
	}

	return attrs;
}

function generatedMockedData(): IExportLogsServiceRequest {
	// Seed and time bounds both derive from the current window bucket, so any
	// calls landing in the same window produce an identical payload.
	const bucket = Math.floor(Date.now() / SEED_WINDOW_MS);
	faker.seed(bucket);
	const windowEnd = bucket * SEED_WINDOW_MS;
	const windowStart = windowEnd - 24 * 60 * 60 * 1000;
	windowRefDate = new Date(windowEnd);

	const resourceCount = faker.number.int({ min: 8, max: 20 });
	const resourceLogs: IResourceLogs[] = new Array(resourceCount)
		.fill(0)
		.map(
			() =>
				({
					resource: {
						droppedAttributesCount: 0,
						attributes: [
							{
								key: "service.namespace",
								value: {
									stringValue: faker.company.buzzNoun(),
								},
							},
							{
								key: "service.name",
								value: {
									stringValue: faker.hacker.noun(),
								},
							},
							{
								key: "service.version",
								value: {
									stringValue: faker.system.semver(),
								},
							},
						],
					},
					scopeLogs: [
						{
							scope: {
								droppedAttributesCount: 0,
								attributes: [
									{
										key: "telemetry.sdk.name",
										value: {
											stringValue:
												"dash0-take-home-assignment",
										},
									},
									{
										key: "telemetry.sdk.language",
										value: {
											stringValue: "nodejs",
										},
									},
									{
										key: "telemetry.sdk.version",
										value: {
											stringValue: "1.0.0",
										},
									},
								],
								name: "mock",
							},
							logRecords: faker.date
								.betweens({
									from: new Date(windowStart),
									to: new Date(windowEnd),
									count: faker.number.int({
										min: 5,
										max: 50,
									}),
								})
								.map((date) => {
									const timeUnixNano = (
										BigInt(date.getTime()) *
										BigInt(1000000)
									).toString();
									const severityNumber = faker.number.int({
										min: 0,
										max: severityTexts.length - 1,
									});
									const logRecord: ILogRecord = {
										timeUnixNano,
										observedTimeUnixNano: timeUnixNano,
										severityNumber,
										severityText:
											severityTexts[severityNumber],
										body: {
											stringValue: generateLogBody(),
										},
										attributes:
											generateLogAttributes(
												severityNumber
											),
										droppedAttributesCount: 0,
									};

									return logRecord;
								}),
						},
					],
				}) satisfies IResourceLogs
		);

	return {
		resourceLogs,
	};
}

export async function OPTIONS() {
	return Response.json(
		{},
		{
			headers: {
				"Access-Control-Allow-Origin": "*",
			},
		}
	);
}
