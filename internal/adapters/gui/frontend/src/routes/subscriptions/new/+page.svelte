<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { z } from 'zod';
  import { appTitle } from '$lib/scaffold-readiness';
  import { saveSubscriptionAndRefresh } from '$lib/stores/subscription';
  import { notificationStore } from '$lib/stores/notification';
  import { listSubscriptionPresets } from '$lib/bindings/forms';
  import Form from '$lib/components/forms/Form.svelte';
  import FormField from '$lib/components/forms/FormField.svelte';
  import TextInput from '$lib/components/forms/TextInput.svelte';
  import NumberInput from '$lib/components/forms/NumberInput.svelte';
  import SelectInput from '$lib/components/forms/SelectInput.svelte';
  import DatePicker from '$lib/components/forms/DatePicker.svelte';
  import Toggle from '$lib/components/forms/Toggle.svelte';
  import type { SubscriptionInput, SubscriptionPresetState } from '$lib/types/forms';

  const subscriptionSchema = z.object({
    presetKey: z.string().optional(),
    provider: z.string().min(1, 'Provider is required'),
    planName: z.string().min(1, 'Plan Name is required'),
    renewalDay: z.number().min(1, 'Renewal day must be between 1 and 31').max(31, 'Renewal day must be between 1 and 31'),
    startsAt: z.string().min(1, 'Start date is required'),
    endsAt: z.string().optional(),
    feeUsd: z.number().min(0, 'Fee cannot be negative'),
    isActive: z.boolean()
  }).refine(data => {
    if (!data.isActive && !data.endsAt) {
      return false;
    }
    return true;
  }, {
    message: "Inactive subscriptions must include an end date",
    path: ["endsAt"]
  }).refine(data => {
    if (data.endsAt && new Date(data.endsAt) < new Date(data.startsAt)) {
      return false;
    }
    return true;
  }, {
    message: "End date must be at or after start date",
    path: ["endsAt"]
  });

  let formData: SubscriptionInput = $state({
    presetKey: '',
    provider: '',
    planName: '',
    renewalDay: 1,
    startsAt: new Date().toISOString().split('T')[0],
    endsAt: '',
    feeUsd: 0,
    isActive: true
  });

  let errors: Record<string, string> = $state({});
  let isSubmitting = $state(false);
  let presets: SubscriptionPresetState[] = $state([]);

  let presetOptions = $derived([
    { value: '', label: 'Custom Plan' },
    ...presets.map(p => ({ value: p.key, label: `${p.provider} - ${p.planName} ($${p.feeUsd})` }))
  ]);

  onMount(async () => {
    try {
      const response = await listSubscriptionPresets();
      presets = response.items || [];
    } catch (error) {
      notificationStore.addNotification({
        id: crypto.randomUUID(),
        title: 'Error',
        body: error instanceof Error ? error.message : 'Failed to load presets',
        kind: 'error',
        severity: 'critical'
      });
    }
  });

  function handlePresetChange(key: string) {
    if (!key) {
      formData.presetKey = '';
      return;
    }
    
    const preset = presets.find(p => p.key === key);
    if (preset) {
      formData.presetKey = preset.key;
      formData.provider = preset.provider;
      formData.planName = preset.planName;
      formData.renewalDay = preset.renewalDay;
      formData.feeUsd = preset.feeUsd;
    }
  }

  $effect(() => {
    handlePresetChange(formData.presetKey);
  });

  async function handleSubmit() {
    const result = subscriptionSchema.safeParse(formData);
    if (!result.success) {
      errors = {};
      result.error.issues.forEach((issue) => {
        if (issue.path[0]) {
          errors[issue.path[0].toString()] = issue.message;
        }
      });
      return;
    }

    errors = {};
    isSubmitting = true;

    try {
      const payload = {
        ...formData,
        startsAt: formData.startsAt,
        endsAt: formData.endsAt ? formData.endsAt : ''
      };
      
      const response = await saveSubscriptionAndRefresh(payload);

      if (response.result.success) {
        notificationStore.addNotification({
          id: crypto.randomUUID(),
          title: 'Success',
          body: 'Subscription saved',
          kind: 'info',
          severity: 'info'
        });
        
        goto('/subscriptions');
      } else {
        notificationStore.addNotification({
          id: crypto.randomUUID(),
          title: 'Error',
          body: response.result.error?.message || 'Failed to save subscription',
          kind: 'error',
          severity: 'critical'
        });
      }
    } catch (error) {
      notificationStore.addNotification({
        id: crypto.randomUUID(),
        title: 'Error',
        body: error instanceof Error ? error.message : 'An unexpected error occurred',
        kind: 'error',
        severity: 'critical'
      });
    } finally {
      isSubmitting = false;
    }
  }

  function handleCancel() {
    goto('/subscriptions');
  }
</script>

<svelte:head>
  <title>{appTitle} - New Subscription</title>
</svelte:head>

<div class="p-8 max-w-3xl mx-auto space-y-8 relative">
  <div>
    <h1 class="text-3xl font-bold text-text">Add Subscription</h1>
    <p class="mt-2 text-text-muted">Create a new API subscription to track recurring costs.</p>
  </div>

  <div class="bg-card border border-panel-border rounded-lg p-6 shadow-sm">
    <Form onsubmit={handleSubmit}>
      <FormField id="presetKey" label="Preset" error={errors.presetKey}>
        <SelectInput
          id="presetKey"
          bind:value={formData.presetKey}
          options={presetOptions}
          error={!!errors.presetKey}
        />
      </FormField>

      <div class="grid grid-cols-2 gap-md">
        <FormField id="provider" label="Provider" error={errors.provider} required>
          <TextInput
            id="provider"
            bind:value={formData.provider}
            placeholder="e.g. openai"
            error={!!errors.provider}
          />
        </FormField>

        <FormField id="planName" label="Plan Name" error={errors.planName} required>
          <TextInput
            id="planName"
            bind:value={formData.planName}
            placeholder="e.g. ChatGPT Plus"
            error={!!errors.planName}
          />
        </FormField>
      </div>

      <div class="grid grid-cols-2 gap-md">
        <FormField id="feeUsd" label="Fee (USD)" error={errors.feeUsd} required>
          <NumberInput
            id="feeUsd"
            bind:value={formData.feeUsd}
            error={!!errors.feeUsd}
            step="0.01"
          />
        </FormField>

        <FormField id="renewalDay" label="Renewal Day (1-31)" error={errors.renewalDay} required>
          <NumberInput
            id="renewalDay"
            bind:value={formData.renewalDay}
            error={!!errors.renewalDay}
            min={1}
            max={31}
          />
        </FormField>
      </div>

      <div class="grid grid-cols-2 gap-md">
        <FormField id="startsAt" label="Starts At" error={errors.startsAt} required>
          <DatePicker
            id="startsAt"
            bind:value={formData.startsAt}
            error={!!errors.startsAt}
          />
        </FormField>

        <FormField id="endsAt" label="Ends At" error={errors.endsAt}>
          <DatePicker
            id="endsAt"
            bind:value={formData.endsAt}
            error={!!errors.endsAt}
          />
        </FormField>
      </div>

      <FormField id="isActive" label="Active Status" error={errors.isActive}>
        <div class="mt-2">
          <div class="flex items-center gap-2">
            <Toggle
              id="isActive"
              bind:checked={formData.isActive}
            />
            <span class="text-sm text-text">{formData.isActive ? 'Active' : 'Inactive'}</span>
          </div>
        </div>
      </FormField>

      <div class="flex justify-end space-x-4 mt-8">
        <button
          type="button"
          class="px-4 py-2 text-sm font-medium text-text-muted hover:text-text transition-colors"
          onclick={handleCancel}
          disabled={isSubmitting}
        >
          Cancel
        </button>
        <button
          type="submit"
          class="px-4 py-2 text-sm font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50"
          disabled={isSubmitting}
        >
          {isSubmitting ? 'Saving...' : 'Save Subscription'}
        </button>
      </div>
    </Form>
  </div>
</div>
