"use client";
import React, { useMemo } from "react";
import {
	Bar,
	BarChart,
	ResponsiveContainer,
	Tooltip,
	XAxis,
	YAxis,
} from "recharts";
import { ReportLog } from "@/types/logs";

interface Props {
	logs: ReportLog[];
}

const INTERVAL = 1000 * 60 * 60 * 24; // Day
const timeLabelFormatter = new Intl.DateTimeFormat();

const LogsHistogram: React.FC<Props> = ({ logs }) => {
	const bins = useMemo(() => {
		// Array is sorted in descending order hence the last element is the first log
		const first = logs[logs.length - 1].time;
		const last = logs[0].time;
		const binsCount = Math.ceil(
			(last.getTime() - first.getTime()) / INTERVAL
		);

		const result = Array(binsCount)
			.fill({})
			.map((_, i) => {
				return {
					time: new Date(first.getTime() + i * INTERVAL),
					count: 0,
				};
			});

		logs.forEach((log) => {
			const index = Math.floor(
				(log.time.getTime() - first.getTime()) / INTERVAL
			);
			result[index].count++;
		});

		return result;
	}, [logs]);

	return (
		<ResponsiveContainer width="100%" height={100}>
			<BarChart
				data={bins}
				margin={{ top: 5, left: 0, bottom: 5, right: 0 }}
				barCategoryGap="1%"
			>
				<XAxis
					dataKey="time"
					tickFormatter={timeLabelFormatter.format}
					hide
				/>
				<YAxis domain={[0, "dataMax"]} hide />
				<Tooltip
					labelFormatter={timeLabelFormatter.format}
					isAnimationActive={false}
				/>
				<Bar dataKey="count" fill="#8884d8" />
			</BarChart>
		</ResponsiveContainer>
	);
};

export default React.memo(LogsHistogram);
