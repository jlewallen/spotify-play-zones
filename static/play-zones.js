/**
 *
 */

class Device extends React.Component {
    render() {
        const { device, onClick } = this.props;

        let classes = "device row";
        if (device.is_active) {
            classes += " active";
        }

        return (
            <div className={classes} onClick={() => onClick()}>
                <div className="name">{device.name}</div>
                <div className="details">{device.type} Vol = {device.volume_percent}</div>
            </div>
        );
    }
}

class PlayZonesPage extends React.Component {
    refresh() {
        return $.getJSON("devices.json").then((data) => {
            return this.setState({
                devices: data
            });
        });
    }

    refreshAndSchedule() {
        this.refresh().then(() => {
            setTimeout(() => {
                this.refreshAndSchedule();
            }, 1000);
        });
    }

    componentWillMount() {
        this.setState( {
            devices: []
        });

        this.refreshAndSchedule();
    }

    selectDevice(device) {
        const payload = JSON.stringify({
            id: device.id
        });
        return $.ajax({
            type: "POST",
            url: "transfer.json",
            dataType: 'json',
            data: payload
        }).then((data) => {
            return this.setState({
                devices: data
            });
        });
    }

    render() {
        const { devices } = this.state;

        return (
            <div>
                {devices.map((d, i) => (<Device key={i} device={d} onClick={() => this.selectDevice(d)} />))}
            </div>
        );
    }
}

var rootComponent = <PlayZonesPage />;
ReactDOM.render(rootComponent, document.getElementById('root'));
