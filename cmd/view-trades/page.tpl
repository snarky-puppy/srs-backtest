<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Awesome go-echarts</title>
    <script src="/static/echarts.min.js"></script>
    <script src="/static/themes/westeros.js"></script>
    <style> .box { justify-content:center; display:flex; flex-wrap:wrap } </style>
</head>

<body>
    <div>
        <a href='/'>home</a>&nbsp;&nbsp;<a href='/?d={{ .Prev }}'>prev</a> <a href='/?d={{ .Next }}'>next</a>
    </div>
    <div class="box">
        <div class="container">
            <div class="item" id="chart" style="width:1200px;height:500px;"></div>
        </div>
        <table border=1>
            <tr>
                <th>#</th>
                <th>Entry Reason</th>
                <th>Entry Time</th>
                <th>Entry Price</th>
                <th>Exit Time</th>
                <th>Exit Price</th>
                <th>Profit</th>
                <th>Exit Reason</th>
            </tr>
            {{range $i, $a := .Signal.Trades}}
            <tr>
                <td>{{ $i }}</td>
                <td>{{ $a.OpenReason }}</td>
                <td>{{ $a.EntryTime.Format "15:04" }}</td>
                <td>{{ $a.EntryPrice }}</td>
                <td>{{ $a.ExitTime.Format "15:04" }}</td>
                <td>{{ $a.ExitPrice }}</td>
                <td>{{ $a.Profit }}</td>
                <td>{{ $a.ExitReason }}</td>
            </tr>
            {{end}}
        </table>
    </div>
    <script type="text/javascript">
        var data = [
            {{- range .Candle }}[{{.Open}}, {{ .Close }}, {{ .Low }}, {{ .High }}],{{- end }}
        ];
        var xData = [
            {{- range .Candle }}"{{ .Timestamp.Format "15:04" }}",{{- end }}
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
    	function fix(timeStr) {
          const [hour, minutes] = timeStr.split(":").map(Number);
          const roundedMinutes = Math.floor(minutes / 5) * 5;
          const roundedTime = `${hour.toString().padStart(2, "0")}:${roundedMinutes.toString().padStart(2, "0")}`;
          return roundedTime;
        }
        var chart = echarts.init(document.getElementById('chart'), 'westeros');
        var option = {
            series: [
                {
                    type: 'candlestick',
                    data: data,
                    itemStyle: {
                        normal: {
                            color: '#ef232a',
                            color0: '#14b143',
                            borderColor: '#ef232a',
                            borderColor0: '#14b143'
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
                                    xAxis: "{{ .Signal.Bar.EndTime.Format "15:04" }}",
                                }
                            ],
                        ]
                    },
                    markPoint: {
                        data: [
                           {{range $i, $a := .Signal.Trades}}
                            {
                                value: "{{ $i }}. {{ $a.Direction }} {{ $a.OpenReason }}",
                                xAxis: fix("{{ $a.EntryTime.Format "15:04" }}"),
                                yAxis: {{ $a.EntryPrice }},
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
                                value: "{{ $i }}. Exit {{ $a.Profit }} {{ $a.ExitReason }}",
                                xAxis: fix("{{ $a.ExitTime.Format "15:04" }}"),
                                yAxis: {{ $a.ExitPrice }},
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
                  lineStyle: {
                    width: 1,
                    opacity: 0.5
                  }
                },
                {
                  name: 'MA20',
                  type: 'line',
                  data: calculateMA(20, data),
                  smooth: true,
                  showSymbol: false,
                  lineStyle: {
                    width: 1,
                    opacity: 0.5
                  }
                },
            ],
            xAxis: {
                data: xData
            },
            animation: true,
            dataZoom: [{
                type: "inside",
                end: 100
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
