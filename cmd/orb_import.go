package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/CircleCI-Public/circleci-cli/api"
	"github.com/CircleCI-Public/circleci-cli/api/graphql"
)

type orbImportPlan struct {
	NewNamespaces           []string
	NewOrbs                 []api.Orb
	NewVersions             []api.OrbVersion
	AlreadyExistingVersions []api.OrbVersion
}

func (o orbImportPlan) isEmpty() bool {
	n := len(o.NewNamespaces)
	n += len(o.NewOrbs)
	n += len(o.NewVersions)
	return n == 0
}

func importOrb(opts orbOptions) error {
	vs, err := versionsToImport(opts)
	if err != nil {
		return err
	}

	plan, err := generateImportPlan(opts, vs)
	if err != nil {
		return err
	}

	displayPlan(os.Stdout, plan)
	if !opts.noPrompt && !plan.isEmpty() && !opts.tty.askUserToConfirm("Are you sure you would like to proceed?") {
		return nil
	}

	return applyPlan(opts, plan)
}

func versionsToImport(opts orbOptions) ([]api.OrbVersion, error) {
	cloudClient := graphql.NewClient("https://circleci.com", "graphql-unstable", "", opts.cfg.Debug)

	if opts.integrationTesting {
		cloudClient = opts.cl
	}

	var orbVersions []api.OrbVersion
	for _, ref := range opts.args {
		if !isNamespace(ref) {
			version, err := api.OrbInfo(cloudClient, ref)
			if err != nil {
				return nil, fmt.Errorf("orb info: %s", err.Error())
			}

			orbVersions = append(orbVersions, *version)
			continue
		}

		// TODO: support an `--all-versions` flag that gets all versions instead of latest version per orb?
		// Note: fetching all orb versions may not be possible. The best we could do is fetch an arbitrarily large number.
		// Otherwise, do some other operation that grabs orb source data from a single namespace.
		obv, err := api.ListNamespaceOrbVersions(cloudClient, ref)
		if err != nil {
			return nil, fmt.Errorf("list namespace orb versions: %s", err.Error())
		}

		orbVersions = append(orbVersions, obv...)
	}

	// Add a "short name" (the name of the orb without the namespace name,
	// needed for API operations)
	for i := range orbVersions {
		err := orbVersions[i].Orb.AddShortName()
		if err != nil {
			return nil, fmt.Errorf("corrupt orb name: %s", err.Error())
		}
	}

	return orbVersions, nil
}

func generateImportPlan(opts orbOptions, orbVersions []api.OrbVersion) (orbImportPlan, error) {
	uniqueNamespaces := map[string]bool{}
	uniqueOrbs := map[string]api.Orb{}

	// Dedupe namespaces and orbs.
	for _, o := range orbVersions {
		ns, orb := o.Orb.Namespace.Name, o.Orb.ShortName
		uniqueNamespaces[ns] = true
		// Use a {orb}:{namespace} key to address cases where orbs have identical names
		// across distinct namespaces.
		uniqueOrbs[orb+":"+ns] = o.Orb
	}

	var plan orbImportPlan
	for ns := range uniqueNamespaces {
		ok, err := api.NamespaceExists(opts.cl, ns)
		if err != nil {
			return orbImportPlan{}, fmt.Errorf("namespace check failed: %s", err.Error())
		}

		if !ok {
			plan.NewNamespaces = append(plan.NewNamespaces, ns)
		}
	}

	for _, orb := range uniqueOrbs {
		ok, err := api.OrbExists(opts.cl, orb.Namespace.Name, orb.ShortName)
		if err != nil {
			return orbImportPlan{}, fmt.Errorf("orb id check failed: %s", err.Error())
		}

		if !ok {
			plan.NewOrbs = append(plan.NewOrbs, orb)
		}
	}

	for _, o := range orbVersions {
		_, err := api.OrbInfo(opts.cl, fmt.Sprintf("%s@%s", o.Orb.Name, o.Version))
		if _, ok := err.(*api.ErrOrbVersionNotExists); ok {
			plan.NewVersions = append(plan.NewVersions, o)
			continue
		}
		if err != nil {
			return orbImportPlan{}, fmt.Errorf("orb info check failed: %s", err.Error())
		}

		plan.AlreadyExistingVersions = append(plan.AlreadyExistingVersions, o)
	}

	return plan, nil
}

func applyPlan(opts orbOptions, plan orbImportPlan) error {
	for _, ns := range plan.NewNamespaces {
		_, err := api.CreateImportedNamespace(opts.cl, ns)
		if err != nil {
			return fmt.Errorf("unable to create '%s' namespace: %s", ns, err.Error())
		}
	}

	for _, o := range plan.NewOrbs {
		_, err := api.CreateImportedOrb(opts.cl, o.Namespace.Name, o.ShortName)
		if err != nil {
			return fmt.Errorf("unable to create '%s' orb: %s", o.Name, err.Error())
		}
	}

	for _, v := range plan.NewVersions {
		resp, err := api.OrbID(opts.cl, v.Orb.Namespace.Name, v.Orb.ShortName)
		if err != nil {
			return fmt.Errorf("unable to get orb info at %s: %s", v.Orb.Name, err.Error())
		}

		_, err = api.OrbImportVersion(opts.cl, v.Source, resp.Orb.ID, v.Version)
		if err != nil {
			return fmt.Errorf("unable to publish '%s@%s' with source: %s", v.Orb.Name, v.Version, err.Error())
		}
	}

	return nil
}

func displayPlan(w io.Writer, plan orbImportPlan) {
	var b strings.Builder
	b.WriteString("The following actions will be performed:\n")

	for _, ns := range plan.NewNamespaces {
		b.WriteString(fmt.Sprintf("  Create namespace '%s'\n", ns))
	}

	for _, o := range plan.NewOrbs {
		b.WriteString(fmt.Sprintf("  Create orb '%s'\n", o.Name))
	}

	for _, v := range plan.NewVersions {
		b.WriteString(fmt.Sprintf("  Import version '%s@%s'\n", v.Orb.Name, v.Version))
	}

	for i, e := range plan.AlreadyExistingVersions {
		if i == 0 {
			b.WriteString("\nThe following orb versions already exist:\n")
		}
		b.WriteString(fmt.Sprintf("  ('%s@%s')\n", e.Orb.Name, e.Version))
	}

	b.WriteString("\n")

	if plan.isEmpty() {
		b.WriteString("Nothing to do!\n")
	}

	fmt.Fprint(w, b.String())
}

func isNamespace(ref string) bool {
	return len(strings.Split(ref, "/")) == 1
}
