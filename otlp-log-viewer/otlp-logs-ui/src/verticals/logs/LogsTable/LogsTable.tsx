import React from "react";
import LogsService from "@/services/LogsService";
import LogRow from "@/verticals/logs/LogRow";
import styles from "./LogsTable.module.css";

const LogsTable: React.FC = async () => {
	const reportLogs = await LogsService.getReportLogs();

	return (
		<div className={styles.logsTable} role="table">
			<div className={styles.header} role="rowgroup">
				<div role="row">
					<span role="columnheader">Severity</span>
					<span role="columnheader">Time</span>
					<span role="columnheader">Body</span>
				</div>
			</div>

			<div className={styles.body} role="rowgroup">
				{reportLogs.map((log, i) => (
					<LogRow log={log} key={log.body + i} />
				))}
			</div>
		</div>
	);
};

export default React.memo(LogsTable);
