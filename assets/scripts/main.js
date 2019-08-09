'use strict';


const logURL = id =>
    <a href={`/build_log/${id}`}>
        log
    </a>;


const log = (status, id) =>
    status === 'FAILURE' ? (
        logURL(id)
    ) : (
        <span>&nbsp;</span>
    );


const tableHead = () =>
    <thead>
    <tr>
        <th>STATUS</th>
        <th>CHANGES</th>
        <th>AUTHOR</th>
        <th>&nbsp;</th>
    </tr>
    </thead>;


const statusDiagram = (state, status, statusText) =>
    state === "running" ? (
        <div>
            <img alt="success" src="static/loader.gif"/>
            <div>{statusText}</div>
        </div>
    ) : (
        status === "SUCCESS" ? (
            <img alt="success" src="static/statusIconSuccess.png"/>
        ) : (
            <img alt="failure" src="static/statusIconFailed.png"/>
        )
    );


class App extends React.Component {
    constructor(props) {
        super(props);

        this.state = {
            builds: [],
        };
        this.environment = 'Prod';
        this.socketHandler();
    }

    socketHandler() {
        this.ws = new WebSocket('ws://' + window.location.host + '/get_info');
        this.ws.onopen = () => {
            console.log('Server connected. Sending the information about selected build.');
            this.ws.send(JSON.stringify({
                status: 'open',
                environment: this.environment
            }));
        };

        this.ws.onmessage = evt => {
            // on receiving a message, add it to the list of messages
            // const message = JSON.parse(evt.data)
            // this.addMessage(message)
            return Promise.resolve(evt.data)
                .then(JSON.parse)
                .then(
                    updated_builds => {
                        this.setState(() => ({
                            builds: updated_builds.build
                        }));
                    }
                )
        };

        this.ws.onclose = () => {
            console.log('disconnected')
        };
    }

    render() {
        return (
            <div className="app">
                <table className="mdl-data-table mdl-js-data-table mdl-shadow--2dp">
                    {tableHead()}
                    <tbody>
                    {
                        this.state.builds.map(({id, state, status, statusText, changes}) => (
                            changes.change.map(function ({username, comment}, i) {
                                    return <tr key={i}>
                                        <td>{statusDiagram(state, status, statusText)}</td>
                                        <td className="left">
                                            <pre>{comment}</pre>
                                        </td>
                                        <td><b>{username}</b></td>
                                        <td>{log(status, id)}
                                        </td>
                                    </tr>
                                }
                            )
                        ))
                    }

                    </tbody>

                </table>
            </div>
        );
    }
}


ReactDOM.render(
    <App/>,
    document.getElementById('root')
);
