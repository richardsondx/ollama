import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import Plugins from "@/components/Plugins";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/plugins")({
  component: PluginsRoute,
});

function PluginsRoute() {
  return (
    <SidebarLayout title="Plugins" sidebar={<AppSidebar current="plugins" />}>
      <Plugins />
    </SidebarLayout>
  );
}
