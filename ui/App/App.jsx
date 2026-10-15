import React, {useCallback, useState} from 'react';

import user from "../api/resources/user";
import Login from "./views/Login";
import {Navigate, Route, Routes} from "react-router";
import Controls from "./views/Controls";
import {BrowserRouter, Outlet} from "react-router-dom";
import Logs from "./views/Logs";
import Saves from "./views/Saves/Saves";
import Layout from "./components/Layout";
import server from "../api/resources/server";
import Mods from "./views/Mods/Mods";
import UserManagement from "./views/UserManagement/UserManagment";
import ServerSettings from "./views/ServerSettings";
import GameSettings from "./views/GameSettings";
import Console from "./views/Console";
import Help from "./views/Help";
import Credentials from "./views/Credentials";
import DashboardPower from "./views/Dashboard/Power";
import DashboardProduction from "./views/Dashboard/Production";
import DashboardScience from "./views/Dashboard/Science";
import socket from "../api/socket";
import {Flash} from "./components/Flash";


// Both guards live at module scope on purpose. Declared inside App they would be
// a new component type on every render, and React would unmount and remount the
// whole route subtree — which refetches every page's data and throws away its
// state whenever the status broadcast arrives (it ticks every 5 seconds for the
// UPS readout).
const ProtectedRoute = ({isAuthenticated}) => {
    if (!isAuthenticated) {
        return <Navigate to="/login" state={{from: window.location.pathname}}/>;
    }
    return <Outlet/>;
}

// Everything under this route is admin-only. A user-role account is sent to the
// dashboard, the widest page it may see, rather than shown a dead link.
const AdminRoute = ({currentUser}) => {
    if (currentUser?.role !== 'admin') {
        return <Navigate to="/dashboard" replace/>;
    }
    return <Outlet/>;
}

const App = () => {

    const [isAuthenticated, setIsAuthenticated] = useState(false);
    const [currentUser, setCurrentUser] = useState(null);
    const [serverStatus, setServerStatus] = useState(null);

    const handleAuthenticationStatus = useCallback(async (user) => {
        if (user?.username) {
            setCurrentUser(user);
            setIsAuthenticated(true);

            const status = await server.status();
            setServerStatus(status);

            socket.emit('server status subscribe');
            socket.on('server_status', status => {
                setServerStatus(JSON.parse(status));
            });
        }
    },[]);

    const handleLogout = useCallback(async () => {
        const loggedOut = await user.logout();
        if (loggedOut) {
            setIsAuthenticated(false);
            setCurrentUser(null);
        }
    }, []);

    return (
        <BrowserRouter>
            <Routes>
                <Route path="login" element={<Login handleLogin={handleAuthenticationStatus}/>}/>

                {/* route with only `element` will cause the proper children to be place in `<Outlet/>` */}
                <Route element={<ProtectedRoute isAuthenticated={isAuthenticated}/> }>
                    <Route element={<Layout handleLogout={handleLogout} serverStatus={serverStatus} currentUser={currentUser} />}>
                        {/* Read-only for the `user` role: console, mods, dashboards, help */}
                        <Route path="mods" element={<Mods serverStatus={serverStatus} currentUser={currentUser}/>}/>
                        <Route path="console" element={<Console serverStatus={serverStatus}/>}/>
                        <Route path="dashboard" element={<Navigate to="/dashboard-power" replace/>}/>
                        <Route path="dashboard-power" element={<DashboardPower/>}/>
                        <Route path="dashboard-production" element={<DashboardProduction/>}/>
                        <Route path="dashboard-science" element={<DashboardScience/>}/>
                        <Route path="help" element={<Help serverStatus={serverStatus}/>}/>

                        {/* Admin only */}
                        <Route element={<AdminRoute currentUser={currentUser}/>}>
                            <Route index element={<Controls serverStatus={serverStatus}/>}/>
                            <Route path="saves" element={<Saves serverStatus={serverStatus}/>}/>
                            <Route path="server-settings" element={<ServerSettings serverStatus={serverStatus}/>}/>
                            <Route path="game-settings" element={<GameSettings serverStatus={serverStatus}/>}/>
                            <Route path="logs" element={<Logs serverStatus={serverStatus}/>}/>
                            <Route path="user-management" element={<UserManagement serverStatus={serverStatus}/>}/>
                            <Route path="credentials" element={<Credentials serverStatus={serverStatus}/>}/>
                        </Route>
                    </Route>
                </Route>
            </Routes>
        </BrowserRouter>
    );
}

export default App;
