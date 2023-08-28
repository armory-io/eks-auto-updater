# eks-auto-updater

Auto updater docker image designed to take a target cluster and update both managed node pools and addon versions to latest releases

## What?

Currently EKS updates are... painful.  This is a tool to auto update a given clusters dependencies like:

- Addons - find the current default version for a set of addons and update the addon version.
- Managed node AMI - TF doesn't see version changes.  This triggers an update of managed node pool
  - CAUTION:  IF YOU DO NOT have pod disruption budgets or other protections in place, this WILL cause downtime on services.  

## How?

To update your nodegroups, run the following command:

```bash
eks-auto-updater nodegroups \
  --cluster-name <cluster-name> \
  --region <region> \
  --role-arn <role-arn> \
  --nodegroup-wait-time <nodegroup-wait-time> \
  --nodegroup-name <nodegroup-name>
```

To update your addons, run the following command:

```bash
eks-auto-updater addons \
  --cluster-name <cluster-name> \
  --region <region> \
  --role-arn <role-arn> \
  --addons <comma-separated-list-of-addons>
```

## Long term goals

- Possibly update the EKS cluster version?
- Auto update multiple node pools vs. a single one and set of addons
- Lookup addons vs. requiring to be passed
- Lookup addon configuration/identifiy differences vs. just overwriting
