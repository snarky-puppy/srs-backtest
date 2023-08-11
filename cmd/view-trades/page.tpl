<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Peak Profits</title>
    <script src="/static/echarts.min.js"></script>
    <script src="/static/themes/westeros.js"></script>
    <style> .box { justify-content:center; display:flex; flex-wrap:wrap } </style>
</head>

<body>
    <div>
        <ul>
            <li><a href='/'>home</a></li>
            <li><a href="/daily">daily</a> </li>
            <li><a href='/?d={{ .Prev }}'>prev</a></li>
            <li><a href='/?d={{ .Next }}'>next</a></li>
        </ul>
    </div>
    <div class="box">
        <div class="container">
            <div class="item" id="chart" style="width:1200px;height:500px;"></div>
        </div>
        <table border=1>
            <tr>
                <th>#</th>
                <th>Size</th>
                <th>Direction</th>
                <th>Stop Pts</th>
                <th>Entry Reason</th>
                <th>Entry Time</th>
                <th>Entry Price</th>
                <th>Exit Time</th>
                <th>Exit Price</th>
                <th>Pts Profit</th>
                <th>Profit</th>
                <th>Exit Reason</th>
            </tr>
            {{range .Trades}}
            <tr>
                <td>{{ .Id }}</td>
                <td>{{ .Size }}</td>
                <td>{{ .Direction }}</td>
                <td>{{ .TrailStopPoints }}</td>
                <td>{{ .OpenReason }}</td>
                <td>{{ .EntryTime.Format "15:04" }}</td>
                <td>{{ .EntryPrice }}</td>
                <td>{{ .ExitTime.Format "15:04" }}</td>
                <td>{{ .ExitPrice }}</td>
                <td>{{ .PointsProfit }}</td>
                <td>{{ .Profit }}</td>
                <td>{{ .ExitReason }}</td>
            </tr>
            {{end}}
            <tr>
                <td colspan=10 align=right>Profit: &nbsp;</td>
                <td> {{ .Profit }}</td>
                <td></td>
            </tr>
        </table>
    </div>
    <script type="text/javascript">
        var data = [
            {{- range .Series }}[{{.Open}}, {{ .Close }}, {{ .Low }}, {{ .High }}],{{- end }}
        ];
        var xData = [
            {{- range .Series }}"{{ .Timestamp.Format "15:04" }}",{{- end }}
        ];
    	function calculateMA(dayCount, data) {
    		var result = [];
    		for (var i = 0, len = data.length; i < len; i++) {
    			if (i < dayCount) {
    				result.push('-');
    				continue;
    			}
    			var sum = 0;
    			for (var j = 0; j < dayCount; j++) {
    			    let v = data[i - j];
    				sum += (v[0] + v[1] + v[2] + v[3])/4;
    			}
    			result.push(+(sum / dayCount).toFixed(3));
    		}
    		return result;
    	}
        const fix = (timeStr) => {
          const [hour, minutes] = timeStr.split(":").map(Number);
          const roundedMinutes = Math.floor(minutes / 5) * 5;
          return `${hour.toString().padStart(2, "0")}:${roundedMinutes.toString().padStart(2, "0")}`;
        }
        var chart = echarts.init(document.getElementById('chart'), 'westeros');
        var option = {
            series: [
                {
                    type: 'candlestick',
                    data: data,
                    itemStyle: {
                        normal: {
                            color0: '#ef232a',
                            color: '#14b143',
                            borderColor0: '#ef232a',
                            borderColor: '#14b143'
                        }
                    },
                    markArea: {
                        silent: true,
                        data: [
                            [
                                {
                                    itemStyle: {
                                        borderColor: "rgba(255, 0, 0, 0.3)",
                                        borderWidth: 1
                                    },
                                    yAxis: {{ .Signal.Bar.High }},
                                    xAxis: "{{ .Signal.Bar.Timestamp.Format "15:04" }}",
                                },
                                {
                                    yAxis: {{ .Signal.Bar.Low }},
                                    xAxis: "{{ .Signal.Bar.EndBar.Format "15:04" }}",
                                }
                            ],
                        ]
                    },
                    markPoint: {
                        data: [
                           {{range .Trades}}
                            {
                                value: "{{ .Id }}. {{ .Direction }} {{ .OpenReason }}",
                                xAxis: fix("{{ .EntryTime.Format "15:04" }}"),
                                yAxis: {{ .EntryPrice }},
                                symbol: "arrow",
                                symbolSize: 10,
                                symbolRotate: 270,
                                itemStyle: {
                                    color: 'blue'
                                },
                                label: {
                                    show: true,
                                    position: "left"
                                },
                            },
                            {
                                value: "{{ .Id }}. Exit {{ .Profit }} {{ .ExitReason }}",
                                xAxis: fix("{{ .ExitTime.Format "15:04" }}"),
                                yAxis: {{ .ExitPrice }},
                                symbol: "arrow",
                                symbolSize: 10,
                                symbolRotate: 90,
                                itemStyle: {
                                    color: 'blue'
                                },
                                label: {
                                    show: true,
                                    position: "right"
                                },
                            },
                            {{end}}
                        ]
                    },
                },
                {
                  name: 'MA5',
                  type: 'line',
                  data: calculateMA(5, data),
                  smooth: true,
                  showSymbol: false,
                  animation: false,
                  lineStyle: {
                    width: 1,
                    opacity: 0.5
                  }
                },
                {
                  name: 'MA25',
                  type: 'line',
                  data: calculateMA(25, data),
                  smooth: true,
                  showSymbol: false,
                  animation: false,
                  lineStyle: {
                    width: 1,
                    opacity: 0.5
                  }
                },
                {
                  name: 'MA50',
                  type: 'line',
                  data: calculateMA(50, data),
                  smooth: true,
                  showSymbol: false,
                  animation: false,
                  lineStyle: {
                    width: 1,
                    opacity: 0.5
                  }
                },
                {{range .Trades}}
                {
                    name: "Trade {{ .Id }} stop",
                    type: "line",
                    smooth: false,
                    connectNulls: false,
                    showSymbol: false,
                    waveAnimation: false,
                    renderLabelForZeroData: false,
                    selectedMode: false,
                    animation: false,
                    data: [
                        {{- range .StopLine }}{{ . }},{{end}}
                    ]
                },
                {{end}}
            ],
            xAxis: {
                data: xData
            },
            animation: true,
            dataZoom: [{
                type: "inside",
                startValue: "08:30",
                endValue: "18:00"
            }, {
                type: "slider",
                end: 100
            }],
            title: {
                text: '{{ .Title }}',
            },
            tooltip: {
                show: true,
                trigger: 'axis',
                triggerOn: 'mousemove|click',
                axisPointer: {
                    type: 'cross',
                    show: false,
                }
            },
            legend: {
                show: true
            },
            yAxis: {
                scale: true,
                splitArea: {
                    show: true
                }
            }
        };
        let action = {"areas":{},"type":""};
        chart.setOption(option);
        chart.dispatchAction(action);
    </script>
</body>
</html>
