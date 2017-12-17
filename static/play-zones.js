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
                <div className="details">{device.type} volume @ {device.volume_percent}</div>
            </div>
        );
    }
}

class Playing extends React.Component {
    render() {
        const { playing } = this.props;

        return (
            <div className="playing row">
                <div className="title">{playing.Name}</div>
                <div className="album">{playing.Album}</div>
                <div className="artists">{playing.Artists.map((a, i) => (<span key={i}>{a}</span>))}</div>
            </div>
        );
    }
}

class PlayZonesPage extends React.Component {
    refresh() {
        return $.getJSON("devices.json").then((data) => {
            return this.setState({
                playing: data.Playing,
                devices: data.Devices
            });
        });
    }

    refreshAndSchedule() {
        this.refresh().then(() => {
            setTimeout(() => {
                this.refreshAndSchedule();
            }, 10000);
        });
    }

    componentWillMount() {
        this.setState({
            playing: {
                Artists: []
            },
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
        const { playing, devices } = this.state;

        return (
            <div>
                <Playing playing={playing} />
                {devices.map((d, i) => (<Device key={i} device={d} onClick={() => this.selectDevice(d)} />))}
            </div>
        );
    }
}

var rootComponent = <PlayZonesPage />;
ReactDOM.render(rootComponent, document.getElementById('root'));
