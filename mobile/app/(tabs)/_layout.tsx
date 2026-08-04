import { SymbolView } from 'expo-symbols';
import { Link, Redirect, Tabs } from 'expo-router';
import { ActivityIndicator, Pressable, View } from 'react-native';

import Colors from '@/constants/Colors';
import { useColorScheme } from '@/components/useColorScheme';
import { useClientOnlyValue } from '@/components/useClientOnlyValue';
import { useAuth } from '@/src/api/AuthContext';

export default function TabLayout() {
  const colorScheme = useColorScheme();
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <View style={{ flex: 1, backgroundColor: '#0A0A0A', justifyContent: 'center' }}>
        <ActivityIndicator color="#0071E3" />
      </View>
    );
  }

  if (!user) {
    return <Redirect href="/login" />;
  }

  return (
    <Tabs
      screenOptions={{
        tabBarActiveTintColor: Colors[colorScheme].tint,
        headerShown: useClientOnlyValue(false, true),
        tabBarStyle: {
          backgroundColor: '#0A0A0A',
          borderTopColor: '#222',
        },
        headerStyle: {
          backgroundColor: '#0A0A0A',
        },
        headerTintColor: '#FFF',
        headerRight: () => (
          <Link href="/modal" asChild>
            <Pressable style={{ marginRight: 16 }}>
              {({ pressed }) => (
                <SymbolView
                  name={{ ios: 'person.circle', android: 'account_circle', web: 'account_circle' }}
                  tintColor="#FFF"
                  size={24}
                  style={{ opacity: pressed ? 0.5 : 1 }}
                />
              )}
            </Pressable>
          </Link>
        ),
      }}>
      <Tabs.Screen
        name="index"
        options={{
          title: 'Hoje',
          tabBarIcon: ({ color }) => (
            <SymbolView
              name={{ ios: 'doc.text', android: 'description', web: 'description' }}
              tintColor={color}
              size={28}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="finance"
        options={{
          title: 'Finanças',
          tabBarIcon: ({ color }) => (
            <SymbolView
              name={{ ios: 'dollarsign.circle', android: 'attach_money', web: 'attach_money' }}
              tintColor={color}
              size={28}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="health"
        options={{
          title: 'Saúde',
          tabBarIcon: ({ color }) => (
            <SymbolView
              name={{ ios: 'heart', android: 'favorite', web: 'favorite' }}
              tintColor={color}
              size={28}
            />
          ),
        }}
      />
    </Tabs>
  );
}
