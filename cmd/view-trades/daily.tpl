<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Peak Profits - daily performance chart</title>
    <script src="/static/echarts.min.js"></script>
    <script src="/static/themes/westeros.js"></script>
    <style> .box { justify-content:center; display:flex; flex-wrap:wrap } </style>
</head>

<body>
    <div>
        <ul>
            <li><a href='/'>home</a></li>
            <li><a href="/daily">daily</a> </li>
            <li><a href='/'>prev</a></li>
            <li><a href='/'>next</a></li>
        </ul>
    </div>
    <div class="box">
        <div class="container">
            <div class="item" id="chart" style="width:1200px;height:500px;"></div>
        </div>
    </div>
    <script type="text/javascript">
        var data = [
            {{- range .Days }}{{.Profit}},{{- end }}
        ];
        var xData = [
            {{- range .Days }}"{{ .Timestamp.Format "02/01" }}",{{- end }}
        ];
        var chart = echarts.init(document.getElementById('chart'), 'westeros');
        var option = {
            series: [
                {
                    type: 'bar',
                    data: data,
                    itemStyle: {
                        normal: {
                            color0: '#ef232a',
                            color: '#14b143',
                            borderColor0: '#ef232a',
                            borderColor: '#14b143'
                        }
                    },
                },
            ],
            xAxis: {
                data: xData
            },
            animation: true,
            title: {
                text: 'daily performance',
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
