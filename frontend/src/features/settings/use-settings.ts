import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useForm, type FieldErrors } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { getSettings, updateSettings } from "@/features/settings/settings-api";
import { settingsSchema, toSettingsDTO, toSettingsForm, type SettingsForm } from "@/features/settings/settings-model";

function collectErrorPaths(errors: FieldErrors, prefix = ""): string[] {
  const paths: string[] = [];
  for (const [key, value] of Object.entries(errors)) {
    if (!value) continue;
    const path = prefix ? `${prefix}.${key}` : key;
    if (typeof value === "object" && value !== null && "message" in value && value.message) {
      paths.push(path);
      continue;
    }
    if (typeof value === "object" && value !== null) {
      paths.push(...collectErrorPaths(value as FieldErrors, path));
    }
  }
  return paths;
}

export function useSettings() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const settingsQuery = useQuery({ queryKey: ["settings"], queryFn: getSettings });
  const form = useForm<SettingsForm>({
    resolver: zodResolver(settingsSchema),
    // Avoid "click save → nothing" when RHF has not yet re-rendered isValid after edits.
    mode: "onSubmit",
    reValidateMode: "onChange",
  });
  const updateMutation = useMutation({
    mutationFn: (config: SettingsForm) => updateSettings(settingsQuery.data?.revision ?? "0", toSettingsDTO(config)),
    onSuccess: (snapshot) => {
      queryClient.setQueryData(["settings"], snapshot);
      void queryClient.invalidateQueries({ queryKey: ["system-info"] });
      form.reset(toSettingsForm(snapshot.config));
      toast.success(t("settings.saved"));
    },
    onError: (error) => toast.error(error instanceof Error ? error.message : t("errors.generic")),
  });

  useEffect(() => {
    if (settingsQuery.data) form.reset(toSettingsForm(settingsQuery.data.config));
  }, [form, settingsQuery.data]);

  const onInvalid = (errors: FieldErrors<SettingsForm>) => {
    const paths = collectErrorPaths(errors);
    const preview = paths.slice(0, 4).join(", ");
    const more = paths.length > 4 ? ` (+${paths.length - 4})` : "";
    toast.error(
      paths.length > 0
        ? t("settings.validationFailed", { fields: `${preview}${more}`, defaultValue: `校验失败：${preview}${more}` })
        : t("settings.validationFailedGeneric", { defaultValue: "表单校验未通过，请检查标红字段" }),
    );
    // Focus first invalid control so errors on other tabs are less silent.
    const first = paths[0];
    if (first) {
      try {
        form.setFocus(first as never);
      } catch {
        // ignore focus failures for nested duration objects
      }
    }
  };

  return {
    form,
    settingsQuery,
    updateMutation,
    onInvalid,
    reset: () => { if (settingsQuery.data) form.reset(toSettingsForm(settingsQuery.data.config)); },
  };
}
