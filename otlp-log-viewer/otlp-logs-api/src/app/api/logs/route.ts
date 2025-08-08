import { faker } from "@faker-js/faker";
import {
	IExportLogsServiceRequest,
	IResourceLogs,
	ILogRecord,
} from "@opentelemetry/otlp-transformer/build/src/logs/internal-types";

export const runtime = "edge";

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

function generatedMockedData(): IExportLogsServiceRequest {
	const resourceLogs: IResourceLogs[] = new Array(10).fill(0).map(
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
								from: new Date("2024-01-01"),
								to: new Date("2024-01-31"),
								count: faker.number.int({
									min: 1,
									max: 10,
								}),
							})
							.map((date) => {
								const timeUnixNano = (
									BigInt(date.getTime()) * BigInt(1000000)
								).toString();
								const severityNumber = faker.number.int({
									min: 0,
									max: severityTexts.length - 1,
								});
								const logRecord: ILogRecord = {
									timeUnixNano,
									observedTimeUnixNano: timeUnixNano,
									severityNumber,
									severityText: severityTexts[severityNumber],
									body: {
										stringValue: faker.git.commitMessage(),
									},
									attributes: [],
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
