import {
	IAnyValue,
	IExportLogsServiceRequest,
} from "@opentelemetry/otlp-transformer";
import { ReportLog } from "@/types/logs";

function otlpLogRecordBodyToString(body?: IAnyValue): string {
	if (!body) {
		return "";
	}

	if (body.stringValue) {
		return body.stringValue;
	}

	const keys = Object.keys(body);
	if (!keys.length) {
		return "";
	}

	const key = keys[0] as keyof IAnyValue;
	const value = body[key];

	switch (key) {
		case "boolValue":
		case "intValue":
		case "doubleValue":
			return String(value);

		// For complex types, we just return an empty string for now
		default:
			return "";
	}
}

/**
 * Aggregates OTLP logs into a list of ReportLog's
 */
export default function otlpLogsToReportLogs(
	otlpLogs: IExportLogsServiceRequest
): ReportLog[] {
	if (!otlpLogs.resourceLogs) {
		return [];
	}

	return otlpLogs.resourceLogs
		.reduce((result, { scopeLogs }) => {
			scopeLogs.forEach(({ logRecords }) => {
				logRecords?.forEach((record) => {
					result.push({
						record,
						body: otlpLogRecordBodyToString(record.body),
						severity: record.severityText,
						time: new Date(Number(record.timeUnixNano) / 1e6),
					});
				});
			});

			return result;
		}, [] as ReportLog[])
		.sort((a, b) => (b.time as any) - (a.time as any));
}
