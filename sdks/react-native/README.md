# @applad/react-native

React Native SDK for [Applad](https://github.com/mittolabs/applad) BaaS.

## Installation

```bash
npm install @applad/react-native
```

### Peer dependencies

```bash
npm install react react-native @react-native-async-storage/async-storage
```

If you haven't already, link AsyncStorage:

```bash
cd ios && pod install
```

## Quick start

Wrap your app with `ApplAdProvider`:

```tsx
import { ApplAdProvider } from '@applad/react-native';

export default function App() {
  return (
    <ApplAdProvider endpoint="https://your-applad-instance.com" projectId="your-project-id">
      <MyApp />
    </ApplAdProvider>
  );
}
```

Then use hooks in any component:

```tsx
import { useAuth, useDatabases } from '@applad/react-native';

function LoginScreen() {
  const auth = useAuth();

  const handleLogin = async () => {
    const session = await auth.createEmailSession('user@example.com', 'password123');
    console.log('Logged in:', session);
  };

  return <Button title="Login" onPress={handleLogin} />;
}

function TodoList() {
  const databases = useDatabases();
  const [todos, setTodos] = useState([]);

  useEffect(() => {
    databases.listDocuments('main', 'todos').then(setTodos);
  }, []);

  return <FlatList data={todos} renderItem={({ item }) => <Text>{item.title}</Text>} />;
}
```

## Services

### Auth

```ts
const auth = useAuth();

// Create account
await auth.createAccount('user@example.com', 'password', { name: 'Jane' });

// Login
await auth.createEmailSession('user@example.com', 'password');

// Anonymous session
await auth.createAnonymousSession();

// Phone auth
await auth.createPhoneSession('+15551234567');
await auth.verifyPhoneOTP(userId, otpCode);

// OAuth (opens device browser)
await auth.openOAuth('google', 'myapp://oauth/success', 'myapp://oauth/failure');

// Get current user
const user = await auth.getAccount();

// Logout
await auth.logout();
```

### Databases

```ts
const databases = useDatabases();

// CRUD
await databases.createDocument('dbId', 'tableId', { title: 'Hello', done: false });
const docs = await databases.listDocuments('dbId', 'tableId', { limit: 25 });
const doc = await databases.getDocument('dbId', 'tableId', 'rowId');
await databases.updateDocument('dbId', 'tableId', 'rowId', { data: { done: true } });
await databases.deleteDocument('dbId', 'tableId', 'rowId');

// Upsert
await databases.upsertDocument('dbId', 'tableId', 'rowId', { title: 'Updated' });
```

### Storage

```ts
const storage = useStorage();

// Upload from image picker result
await storage.uploadFile('bucketId', 'file:///path/to/photo.jpg', 'photo.jpg', 'image/jpeg');

// List files
const files = await storage.listFiles('bucketId');

// Get preview URL (for Image component)
const url = storage.getFilePreview('bucketId', 'fileId', { width: 200, height: 200 });
```

### Realtime

```ts
const realtime = useRealtime();

// Connect
await realtime.connect();

// Subscribe
const unsub = realtime.subscribe('databases.main.tables.todos.rows', (event) => {
  console.log('Change:', event.payload);
});

// Cleanup
unsub();
realtime.disconnect();
```

### Functions

```ts
const functions = useFunctions();

const result = await functions.invoke('my-function-id', { key: 'value' });
const executions = await functions.listExecutions('my-function-id');
```

### Flags

```ts
const flags = useFlags();

const flag = await flags.getFlag('dark-mode');
const all = await flags.getAllFlags({ userId: '123' });
const evaluated = await flags.evaluateFlag('beta-feature', { plan: 'pro' });
```

## Direct client usage (without hooks)

```ts
import { ApplAdClient, Auth, Databases } from '@applad/react-native';

const client = new ApplAdClient({
  endpoint: 'https://your-applad-instance.com',
  projectId: 'your-project-id',
});

const auth = new Auth(client);
const databases = new Databases(client);

await auth.createEmailSession('user@example.com', 'password');
const docs = await databases.listDocuments('dbId', 'tableId');
```

## Session persistence

Sessions are automatically persisted to AsyncStorage. After `createEmailSession` or `createAnonymousSession`, the JWT is stored and restored on app restart. Call `auth.logout()` or `client.clearSession()` to clear it.

## License

MIT
