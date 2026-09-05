import React, { useEffect, useState } from 'react';
import { SnapTradeConnect } from '../components/SnaptradeConnect';
// Assuming you have a PlaidConnect component, import it here
// import { PlaidConnect } from '../components/PlaidConnect';

// Define the shape of the data returned by our unified Go endpoint
interface Connection {
    broker: 'PLAID' | 'SNAPTRADE';
    institution_name: string;
    total_accounts: number;
    is_active: boolean;
}

export const ManageConnectionsView: React.FC = () => {
    const [connections, setConnections] = useState<Connection[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Fetch all unified connections from the backend
    const fetchConnections = async () => {
        setIsLoading(true);
        setError(null);
        try {
            const response = await fetch('/api/v1/connections');
            if (!response.ok) throw new Error('Failed to load connections');
            const data = await response.json();
            setConnections(data);
        } catch (err) {
            console.error(err);
            setError('Could not load your connected accounts.');
        } finally {
            setIsLoading(false);
        }
    };

    // Load connections on component mount
    useEffect(() => {
        fetchConnections();
    }, []);

    // Helper function to render a single connection card
    const renderConnectionCard = (conn: Connection, idx: number) => (
        <div 
            key={`${conn.broker}-${conn.institution_name}-${idx}`} 
            className="flex items-center justify-between p-4 border rounded-lg bg-white shadow-sm"
        >
            <div className="flex flex-col">
                <span className="font-semibold text-gray-900">{conn.institution_name}</span>
                <span className="text-sm text-gray-500">
                    {conn.total_accounts} {conn.total_accounts === 1 ? 'Account' : 'Accounts'}
                </span>
            </div>
            <div className="flex items-center gap-3">
                {/* Visual indicator for connection health based on the is_active flag from our ghost-buster logic */}
                <span className={`px-2 py-1 text-xs font-medium rounded-full ${
                    conn.is_active ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                }`}>
                    {conn.is_active ? 'Connected' : 'Action Required'}
                </span>
                
                {/* Future enhancement: Add disconnect or repair handlers here */}
                <button className="text-sm text-gray-400 hover:text-red-500 transition-colors">
                    Disconnect
                </button>
            </div>
        </div>
    );

    // Separate connections by tier for the UI
    const premiumConnections = connections.filter(c => c.broker === 'SNAPTRADE');
    const basicConnections = connections.filter(c => c.broker === 'PLAID');

    return (
        <div className="max-w-4xl mx-auto p-6 space-y-8">
            <div>
                <h1 className="text-2xl font-bold text-gray-900">Manage Connections</h1>
                <p className="text-gray-600 mt-1">Connect your banks and brokerages to sync your portfolio.</p>
            </div>

            {error && (
                <div className="p-4 bg-red-50 text-red-700 rounded-md">
                    {error}
                </div>
            )}

            {/* ==========================================
                PREMIUM TIER: SNAPTRADE
            ========================================== */}
            <section className="space-y-4">
                <div className="flex items-center justify-between border-b pb-2">
                    <div>
                        <h2 className="text-lg font-semibold text-gray-800">Premium Brokerages</h2>
                        <p className="text-sm text-gray-500">High-performance sync for live trading and options.</p>
                    </div>
                    {/* Render the wrapper component we just built */}
                    <SnapTradeConnect onSuccess={fetchConnections} />
                </div>

                {isLoading ? (
                    <div className="animate-pulse h-16 bg-gray-100 rounded-lg"></div>
                ) : premiumConnections.length > 0 ? (
                    <div className="grid gap-3">
                        {premiumConnections.map(renderConnectionCard)}
                    </div>
                ) : (
                    <div className="p-6 text-center border-2 border-dashed rounded-lg text-gray-500">
                        No premium brokerages connected yet.
                    </div>
                )}
            </section>

            {/* ==========================================
                BASIC TIER: PLAID
            ========================================== */}
            <section className="space-y-4 pt-4">
                <div className="flex items-center justify-between border-b pb-2">
                    <div>
                        <h2 className="text-lg font-semibold text-gray-800">Basic Connections</h2>
                        <p className="text-sm text-gray-500">Standard sync for read-only banking and cash flow.</p>
                    </div>
                    {/* Placeholder for Plaid connection button */}
                    <button className="px-4 py-2 bg-gray-100 text-gray-700 rounded-md font-medium hover:bg-gray-200 transition-colors">
                        Add Bank Account
                    </button>
                </div>

                {isLoading ? (
                    <div className="animate-pulse h-16 bg-gray-100 rounded-lg"></div>
                ) : basicConnections.length > 0 ? (
                    <div className="grid gap-3">
                        {basicConnections.map(renderConnectionCard)}
                    </div>
                ) : (
                    <div className="p-6 text-center border-2 border-dashed rounded-lg text-gray-500">
                        No basic bank accounts connected.
                    </div>
                )}
            </section>
        </div>
    );
};