import React from "react";
import LogsHistogram from "./LogsHistogram";
import LogsService from "@/services/LogsService";

type Props = Omit<React.ComponentProps<typeof LogsHistogram>, "logs">;

const LogsHistogramContainer: React.FC<Props> = async (props) => {
	const reportLogs = await LogsService.getReportLogs();

	return <LogsHistogram {...props} logs={reportLogs} />;
};

export default React.memo(LogsHistogramContainer);
